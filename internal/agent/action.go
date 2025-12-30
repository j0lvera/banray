package agent

import "fmt"

// ActionType represents the type of action to execute.
type ActionType string

const (
	ActionTypeBash ActionType = "bash"
)

// Action represents a parsed command to execute.
type Action struct {
	Type    ActionType
	Command string
}

// String returns a string representation of the action for debugging.
func (a Action) String() string {
	return fmt.Sprintf("%s: %s", a.Type, a.Command)
}

// Output represents the result of command execution.
type Output struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
}

// String formats the output for display to the LLM.
func (o Output) String() string {
	result := o.Stdout
	if o.Stderr != "" {
		result += "\nstderr: " + o.Stderr
	}
	if o.ExitCode != 0 {
		result += fmt.Sprintf("\nexit code: %d", o.ExitCode)
	}
	return result
}
