package agent

import "fmt"

// TerminationReason indicates why the agent stopped.
type TerminationReason string

const (
	ReasonComplete  TerminationReason = "complete"
	ReasonStepLimit TerminationReason = "step_limit"
)

// TerminatingErr signals the agent should stop the loop.
type TerminatingErr struct {
	Reason TerminationReason
	Output string // Optional final output
}

func (e *TerminatingErr) Error() string {
	return fmt.Sprintf("terminating: %s", e.Reason)
}

// ProcessErrType indicates the type of recoverable error.
type ProcessErrType string

const (
	ProcessErrFormat    ProcessErrType = "format"
	ProcessErrTimeout   ProcessErrType = "timeout"
	ProcessErrExecution ProcessErrType = "execution"
)

// ProcessErr signals a recoverable error. The agent should add
// feedback to messages and continue the loop.
type ProcessErr struct {
	Type    ProcessErrType
	Message string // Feedback to add to conversation
}

func (e *ProcessErr) Error() string {
	return fmt.Sprintf("process error [%s]: %s", e.Type, e.Message)
}
