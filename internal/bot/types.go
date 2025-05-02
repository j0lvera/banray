package bot

import "context"

// Response represents a response from an AI service
type Response struct {
	Content string
}

// Service defines the interface for AI services that the bot can use
type Service interface {
	Reply(ctx context.Context, prompt string, history []Message) (Response, error)
}
