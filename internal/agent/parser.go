package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// Parser extracts executable actions from LLM responses.
type Parser interface {
	ParseAction(response string) (Action, error)
}

// commandRegex matches ```bash\n...\n``` code blocks
var commandRegex = regexp.MustCompile("(?s)```bash\\s*\\n(.*?)\\n```")

// BashParser extracts bash commands from markdown code blocks.
type BashParser struct{}

// NewBashParser creates a new bash command parser.
func NewBashParser() *BashParser {
	return &BashParser{}
}

// ParseAction extracts a single bash command from the response.
func (p *BashParser) ParseAction(response string) (Action, error) {
	matches := commandRegex.FindAllStringSubmatch(response, -1)

	if len(matches) == 0 {
		return Action{}, &ProcessErr{
			Type:    ProcessErrFormat,
			Message: "No bash command found. If the task is complete, respond with TASK_COMPLETE. Otherwise, provide exactly one command in ```bash``` block.",
		}
	}

	if len(matches) > 1 {
		return Action{}, &ProcessErr{
			Type:    ProcessErrFormat,
			Message: fmt.Sprintf("Found %d commands, expected exactly one. Please provide a single command in ```bash``` block.", len(matches)),
		}
	}

	command := strings.TrimSpace(matches[0][1])
	if command == "" {
		return Action{}, &ProcessErr{
			Type:    ProcessErrFormat,
			Message: "Empty command in bash block. Please provide a valid command.",
		}
	}

	return Action{
		Type:    ActionTypeBash,
		Command: command,
	}, nil
}
