package agent

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Querier sends messages to an LLM and receives responses.
type Querier interface {
	Query(ctx context.Context, messages []Message) (string, error)
}

// OpenAIQuerier implements Querier using the OpenAI-compatible API.
type OpenAIQuerier struct {
	client llms.Model
}

// NewOpenAIQuerier creates a new OpenAI-compatible querier.
func NewOpenAIQuerier(apiKey, baseURL, model string) (*OpenAIQuerier, error) {
	client, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL(baseURL),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &OpenAIQuerier{client: client}, nil
}

// Query sends messages to the LLM and returns the response.
func (q *OpenAIQuerier) Query(ctx context.Context, messages []Message) (string, error) {
	llmMessages := make([]llms.MessageContent, 0, len(messages))

	for _, msg := range messages {
		var msgType llms.ChatMessageType
		switch msg.Role {
		case RoleSystem:
			msgType = llms.ChatMessageTypeSystem
		case RoleUser:
			msgType = llms.ChatMessageTypeHuman
		case RoleAssistant:
			msgType = llms.ChatMessageTypeAI
		default:
			continue
		}
		llmMessages = append(llmMessages, llms.TextParts(msgType, msg.Content))
	}

	resp, err := q.client.GenerateContent(ctx, llmMessages)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from model")
	}

	return resp.Choices[0].Content, nil
}
