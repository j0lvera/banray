package agent

import "fmt"

// Role represents the sender of a message in the conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single message in the conversation history.
type Message struct {
	Role    Role
	Content string
}

// String returns a string representation of the message for debugging.
func (m Message) String() string {
	return fmt.Sprintf("%s: %s", m.Role, m.Content)
}
