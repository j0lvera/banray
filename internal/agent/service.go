package agent

import (
	"context"
	"fmt"

	"github.com/j0lvera/banray/internal/bot"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Service implements the bot.Service interface using langchain-go
type Service struct {
	client llms.Model
}

func NewService(apiKey, baseURL, model string) (*Service, error) {
	client, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL(baseURL),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &Service{
		client: client,
	}, nil
}

// Reply implements the bot.Service interface
func (s *Service) Reply(
	ctx context.Context, prompt string, history []bot.Message,
) (bot.Response, error) {
	// Build messages starting with system prompt
	msgs := []llms.MessageContent{
		llms.TextParts(
			llms.ChatMessageTypeSystem,
			"Provide brief, concise responses with a friendly and human tone.",
		),
	}

	// Add conversation history
	for _, msg := range history {
		// Skip the current prompt which will be added separately
		if msg.Role == "user" && msg.Content == prompt {
			continue
		}

		var msgType llms.ChatMessageType
		switch msg.Role {
		case "user":
			msgType = llms.ChatMessageTypeHuman
		case "assistant":
			msgType = llms.ChatMessageTypeAI
		default:
			continue
		}

		msgs = append(msgs, llms.TextParts(msgType, msg.Content))
	}

	// Add the current prompt
	msgs = append(msgs, llms.TextParts(llms.ChatMessageTypeHuman, prompt))

	resp, err := s.client.GenerateContent(ctx, msgs)
	if err != nil {
		return bot.Response{}, fmt.Errorf("ai service generate error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return bot.Response{}, fmt.Errorf("no choices returned from model")
	}

	return bot.Response{
		Content: resp.Choices[0].Content,
	}, nil
}
