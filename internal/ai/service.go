package ai

import (
	"context"
	"fmt"

	"github.com/j0lvera/banray/internal/bot"
	"github.com/ollama/ollama/api"
)

// Service implements the bot.Service interface
type Service struct {
	client *api.Client
	model  string
}

func NewService(model string) (*Service, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	if client == nil {
		return nil, fmt.Errorf("AI client is nil after initialization")
	}

	return &Service{
		client: client,
		model:  model,
	}, nil
}

// Reply implements the bot.Service interface
func (s *Service) Reply(
	ctx context.Context, prompt string,
) (bot.Response, error) {
	msgs := []api.Message{
		{
			Role:    "system",
			Content: "Provide very brief, concise responses",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	streamFalse := false
	req := &api.ChatRequest{
		Model:    s.model,
		Messages: msgs,
		Stream:   &streamFalse,
	}

	var result bot.Response

	resFn := func(resp api.ChatResponse) error {
		result.Content = resp.Message.Content
		return nil
	}

	if err := s.client.Chat(ctx, req, resFn); err != nil {
		return bot.Response{}, fmt.Errorf("ai service chat error: %w", err)
	}

	return result, nil
}
