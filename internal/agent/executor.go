package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

// Executor runs actions in the environment.
type Executor interface {
	Execute(ctx context.Context, action Action) (Output, error)
}

// CommandValidator checks if a command is safe to execute.
type CommandValidator interface {
	Validate(command string) error
}

// BlocklistValidator blocks commands matching dangerous patterns.
type BlocklistValidator struct {
	patterns []*regexp.Regexp
}

// DefaultBlockedPatterns contains patterns for dangerous commands.
var DefaultBlockedPatterns = []string{
	`rm\s+-[rf]*\s+/`,           // rm -rf / or rm -r / or rm -f /
	`rm\s+-[rf]*\s+\*`,          // rm -rf * or similar
	`rm\s+-[rf]*\s+~`,           // rm -rf ~
	`>\s*/dev/sd`,               // writing to disk devices
	`mkfs`,                      // formatting filesystems
	`dd\s+if=.*/dev/`,           // dd from devices
	`dd\s+of=.*/dev/`,           // dd to devices
	`chmod\s+777\s+/`,           // chmod 777 on root
	`chown\s+-R\s+.*\s+/`,       // recursive chown on root
	`curl.*\|\s*(ba)?sh`,        // curl | sh (pipe to shell)
	`wget.*\|\s*(ba)?sh`,        // wget | sh
	`:\(\)\{\s*:\|:&\s*\};:`,    // fork bomb
	`/dev/null\s*>\s*/etc/`,     // overwriting /etc files
	`>\s*/etc/passwd`,           // overwriting passwd
	`>\s*/etc/shadow`,           // overwriting shadow
	`shutdown`,                  // system shutdown
	`reboot`,                    // system reboot
	`init\s+0`,                  // system halt
	`halt`,                      // system halt
	`poweroff`,                  // power off
}

// NewBlocklistValidator creates a validator with the given patterns.
func NewBlocklistValidator(patterns []string) (*BlocklistValidator, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return &BlocklistValidator{patterns: compiled}, nil
}

// NewDefaultBlocklistValidator creates a validator with default dangerous patterns.
func NewDefaultBlocklistValidator() *BlocklistValidator {
	v, _ := NewBlocklistValidator(DefaultBlockedPatterns)
	return v
}

// Validate checks if the command matches any blocked pattern.
func (v *BlocklistValidator) Validate(command string) error {
	for _, re := range v.patterns {
		if re.MatchString(command) {
			return &ProcessErr{
				Type:    ProcessErrExecution,
				Message: fmt.Sprintf("Command blocked for safety: matches pattern %q. Please use a safer alternative.", re.String()),
			}
		}
	}
	return nil
}

// BashExecutor executes bash commands with timeout and validation support.
type BashExecutor struct {
	timeout    time.Duration
	workingDir string
	validator  CommandValidator
}

// BashExecutorOption configures a BashExecutor.
type BashExecutorOption func(*BashExecutor)

// WithTimeout sets the command timeout.
func WithTimeout(d time.Duration) BashExecutorOption {
	return func(e *BashExecutor) {
		e.timeout = d
	}
}

// WithWorkingDir sets the working directory for commands.
func WithWorkingDir(dir string) BashExecutorOption {
	return func(e *BashExecutor) {
		e.workingDir = dir
	}
}

// WithValidator sets a custom command validator.
func WithValidator(v CommandValidator) BashExecutorOption {
	return func(e *BashExecutor) {
		e.validator = v
	}
}

// WithoutValidation disables command validation (use with caution).
func WithoutValidation() BashExecutorOption {
	return func(e *BashExecutor) {
		e.validator = nil
	}
}

// NewBashExecutor creates a new bash command executor.
func NewBashExecutor(opts ...BashExecutorOption) *BashExecutor {
	e := &BashExecutor{
		timeout:   30 * time.Second,
		validator: NewDefaultBlocklistValidator(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute runs a bash command and returns the output.
func (e *BashExecutor) Execute(ctx context.Context, action Action) (Output, error) {
	if action.Type != ActionTypeBash {
		return Output{}, fmt.Errorf("unsupported action type: %s", action.Type)
	}

	// Validate command before execution
	if e.validator != nil {
		if err := e.validator.Validate(action.Command); err != nil {
			return Output{}, err
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", action.Command)

	if e.workingDir != "" {
		cmd.Dir = e.workingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := Output{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		// Check if it was a timeout
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			output.TimedOut = true
			return output, &ProcessErr{
				Type:    ProcessErrTimeout,
				Message: fmt.Sprintf("Command timed out after %s. Partial output:\n%s", e.timeout, output.String()),
			}
		}

		// Get exit code if available
		if exitErr, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitErr.ExitCode()
		}

		return output, &ProcessErr{
			Type:    ProcessErrExecution,
			Message: fmt.Sprintf("Command failed: %s\nOutput:\n%s", err.Error(), output.String()),
		}
	}

	return output, nil
}
