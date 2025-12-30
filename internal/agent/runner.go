package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const completionMarker = "TASK_COMPLETE"

// RunnerConfig holds configuration for the agent runner.
type RunnerConfig struct {
	MaxSteps         int           // Maximum number of steps before stopping
	CommandTimeout   time.Duration // Timeout for each command
	WorkingDir       string        // Working directory for commands
	SystemPrompt     string        // Base system prompt
	ContextThreshold int           // Character threshold to trigger summarization (0 = disabled)
}

// DefaultRunnerConfig returns a sensible default configuration.
func DefaultRunnerConfig() RunnerConfig {
	return RunnerConfig{
		MaxSteps:         10,
		CommandTimeout:   30 * time.Second,
		SystemPrompt:     DefaultAgentSystemPrompt,
		ContextThreshold: 8000, // Summarize when context exceeds 8K chars
	}
}

// DefaultAgentSystemPrompt is the system prompt for the bash agent.
const DefaultAgentSystemPrompt = `You are an autonomous agent with bash access. You can execute commands to accomplish tasks.

RULES:
1. Respond with exactly ONE bash command in a code block like this:
` + "```bash" + `
your command here
` + "```" + `

2. After each command, you'll see the output. Use it to decide your next action.

3. When the task is complete, run:
` + "```bash" + `
echo "TASK_COMPLETE"
echo "Your final summary here"
` + "```" + `

4. Be concise. Execute commands, observe results, iterate.

5. If a command fails, try an alternative approach.

6. You have access to common tools: curl, jq, python3, node, etc.`

// Runner orchestrates the agentic loop.
type Runner struct {
	config   RunnerConfig
	querier  Querier
	parser   Parser
	executor Executor
	logger   *zerolog.Logger
	output   io.Writer

	messages []Message
	step     int
	userTask string // Original user request, used for summarization context
}

// NewRunner creates a new agent runner.
func NewRunner(
	config RunnerConfig,
	querier Querier,
	logger *zerolog.Logger,
) *Runner {
	var executorOpts []BashExecutorOption
	if config.CommandTimeout > 0 {
		executorOpts = append(executorOpts, WithTimeout(config.CommandTimeout))
	}
	if config.WorkingDir != "" {
		executorOpts = append(executorOpts, WithWorkingDir(config.WorkingDir))
	}

	return &Runner{
		config:   config,
		querier:  querier,
		parser:   NewBashParser(),
		executor: NewBashExecutor(executorOpts...),
		logger:   logger,
		output:   io.Discard,
		messages: []Message{},
	}
}

// WithOutput sets the output writer for command output streaming.
func (r *Runner) WithOutput(w io.Writer) *Runner {
	r.output = w
	return r
}

// RunResult contains the final output from a run.
type RunResult struct {
	Response   string        // Final response/summary
	Messages   []Message     // Full conversation history
	Steps      int           // Number of steps taken
	TokenUsage TokenUsage    // Aggregated token usage
	Reason     TerminationReason
}

// TokenUsage aggregates token counts across the run.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// Run executes the agent loop until completion or error.
// history should contain previous conversation messages (excluding system prompt and current user message).
func (r *Runner) Run(ctx context.Context, history []Message, userPrompt string) (RunResult, error) {
	result := RunResult{}

	// Initialize conversation
	r.messages = []Message{}
	r.step = 0

	// Store user task for summarization context
	r.userTask = userPrompt

	// Add system prompt
	r.addMessage(RoleSystem, r.config.SystemPrompt)

	// Add conversation history (previous messages from session)
	for _, msg := range history {
		r.addMessage(msg.Role, msg.Content)
	}

	// Add current user message
	r.addMessage(RoleUser, userPrompt)

	r.logger.Info().
		Int("max_steps", r.config.MaxSteps).
		Msg("Starting agent loop")

	var lastResponse string

	// Main loop
	for r.step = 0; r.step < r.config.MaxSteps; r.step++ {
		r.logger.Info().
			Int("step", r.step+1).
			Int("context_size", r.contextSize()).
			Msg("Starting step")

		stepResult, err := r.Step(ctx)
		if err != nil {
			var termErr *TerminatingErr
			var procErr *ProcessErr

			if errors.As(err, &termErr) {
				// Clean exit - task complete or limit reached
				r.logger.Info().
					Str("reason", string(termErr.Reason)).
					Msg("Agent terminated")
				result.Response = termErr.Output
				result.Reason = termErr.Reason
				result.Messages = r.messages
				result.Steps = r.step + 1
				return result, nil
			}

			if errors.As(err, &procErr) {
				// Recoverable - add feedback and continue
				r.logger.Warn().
					Str("type", string(procErr.Type)).
					Str("message", procErr.Message).
					Msg("Process error, continuing")
				r.addMessage(RoleUser, procErr.Message)
				continue
			}

			// Unrecoverable error
			r.logger.Error().Err(err).Msg("Unrecoverable error")
			return result, err
		}

		// Accumulate token usage
		result.TokenUsage.InputTokens += stepResult.InputTokens
		result.TokenUsage.OutputTokens += stepResult.OutputTokens
		result.TokenUsage.TotalTokens += stepResult.TotalTokens

		lastResponse = stepResult.Response
	}

	// Step limit reached
	r.logger.Warn().
		Int("max_steps", r.config.MaxSteps).
		Msg("Step limit reached")

	result.Response = lastResponse
	result.Reason = ReasonStepLimit
	result.Messages = r.messages
	result.Steps = r.step
	return result, &TerminatingErr{Reason: ReasonStepLimit}
}

// StepResult contains the output from a single step.
type StepResult struct {
	Response     string
	Command      string
	Output       Output
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// Step performs a single iteration of the agent loop.
func (r *Runner) Step(ctx context.Context) (StepResult, error) {
	if err := ctx.Err(); err != nil {
		return StepResult{}, fmt.Errorf("context cancelled: %w", err)
	}

	r.logger.Debug().Msg("Querying model")

	// 1. Query the model
	queryResult, err := r.querier.Query(ctx, r.messages)
	if err != nil {
		r.logger.Error().Err(err).Msg("Query failed")
		return StepResult{}, fmt.Errorf("query failed: %w", err)
	}

	r.logger.Debug().
		Int("response_length", len(queryResult.Content)).
		Int("input_tokens", queryResult.InputTokens).
		Int("output_tokens", queryResult.OutputTokens).
		Msg("Got response")

	result := StepResult{
		Response:     queryResult.Content,
		InputTokens:  queryResult.InputTokens,
		OutputTokens: queryResult.OutputTokens,
		TotalTokens:  queryResult.TotalTokens,
	}

	// 2. Parse action from response
	action, err := r.parser.ParseAction(queryResult.Content)
	if err != nil {
		r.logger.Debug().Err(err).Msg("Failed to parse action")
		return result, err
	}
	result.Command = action.Command

	// 3. Add assistant message before execution
	r.addMessage(RoleAssistant, queryResult.Content)

	// 4. Execute the command and stream output
	fmt.Fprintf(r.output, "$ %s\n", action.Command)

	r.logger.Info().
		Str("command", action.Command).
		Msg("Executing command")

	output, err := r.executor.Execute(ctx, action)
	result.Output = output

	if err != nil {
		r.logger.Warn().Err(err).Msg("Command execution failed")
		return result, err
	}

	// Print output (skip if it's just the completion marker)
	if !r.isTaskComplete(output) && strings.TrimSpace(output.Stdout) != "" {
		fmt.Fprintln(r.output, output.Stdout)
	}

	r.logger.Debug().
		Int("output_length", len(output.String())).
		Int("exit_code", output.ExitCode).
		Msg("Command completed")

	// 5. Check for completion signal in command output
	if r.isTaskComplete(output) {
		r.logger.Info().Msg("Task complete signal in output")
		finalOutput := r.extractFinalOutput(output)
		return result, &TerminatingErr{
			Reason: ReasonComplete,
			Output: finalOutput,
		}
	}

	// 6. Format observation (with optional summarization)
	feedback := r.formatObservation(output)

	// 7. Summarize if context is getting too large
	if r.shouldSummarize(len(feedback)) {
		summarized, err := r.summarizeOutput(ctx, feedback)
		if err != nil {
			r.logger.Warn().Err(err).Msg("Failed to summarize, using truncated output")
		} else {
			feedback = summarized
		}
	}

	// 8. Add execution result as user message
	r.addMessage(RoleUser, feedback)

	return result, nil
}

// isTaskComplete checks if the command output starts with the completion signal.
func (r *Runner) isTaskComplete(output Output) bool {
	firstLine := strings.SplitN(strings.TrimSpace(output.Stdout), "\n", 2)[0]
	return strings.TrimSpace(firstLine) == completionMarker
}

// extractFinalOutput returns everything after the completion marker.
func (r *Runner) extractFinalOutput(output Output) string {
	parts := strings.SplitN(output.Stdout, "\n", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// formatObservation formats command output for the LLM.
func (r *Runner) formatObservation(output Output) string {
	if strings.TrimSpace(output.Stdout) == "" && output.ExitCode == 0 {
		return "(no output)"
	}

	result := output.Stdout

	// Truncate long output to keep context manageable
	const maxLen = 3000
	if len(result) > maxLen {
		head := result[:maxLen/2]
		tail := result[len(result)-maxLen/2:]
		result = head + "\n\n[... output truncated ...]\n\n" + tail
	}

	// Add exit code if non-zero
	if output.ExitCode != 0 {
		result = fmt.Sprintf("[exit code: %d]\n%s", output.ExitCode, result)
	}

	return result
}

// addMessage appends a message to the conversation history.
func (r *Runner) addMessage(role Role, content string) {
	r.messages = append(r.messages, Message{
		Role:    role,
		Content: content,
	})
	r.logger.Debug().
		Str("role", string(role)).
		Int("content_length", len(content)).
		Msg("Message added")
}

// Messages returns the current conversation history.
func (r *Runner) Messages() []Message {
	return r.messages
}

// contextSize returns the total character count of all messages.
func (r *Runner) contextSize() int {
	total := 0
	for _, msg := range r.messages {
		total += len(msg.Content)
	}
	return total
}

// shouldSummarize returns true if context is large and output is substantial.
func (r *Runner) shouldSummarize(outputLen int) bool {
	if r.config.ContextThreshold <= 0 {
		return false
	}
	// Summarize if context exceeds threshold and output is substantial (>500 chars)
	return r.contextSize() > r.config.ContextThreshold && outputLen > 500
}

// summarizeOutput asks the model to extract relevant information from large output.
func (r *Runner) summarizeOutput(ctx context.Context, output string) (string, error) {
	prompt := []Message{
		{
			Role: RoleSystem,
			Content: "You are a data extraction assistant. Extract only the specific information needed to answer the user's question. Be concise and preserve exact values (especially dollar amounts).",
		},
		{
			Role: RoleUser,
			Content: fmt.Sprintf("User's question: %s\n\nCommand output:\n%s\n\nExtract only the relevant data needed to answer the question:", r.userTask, output),
		},
	}

	r.logger.Info().
		Int("output_length", len(output)).
		Msg("Summarizing large output")

	result, err := r.querier.Query(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarization failed: %w", err)
	}

	r.logger.Debug().
		Int("original_length", len(output)).
		Int("summarized_length", len(result.Content)).
		Msg("Output summarized")

	return "[Summarized] " + result.Content, nil
}
