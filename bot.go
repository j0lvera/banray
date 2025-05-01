package main

import (
	"context"
	"fmt"
	"strings"

	tbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/kelseyhightower/envconfig"
	"github.com/ollama/ollama/api"
)

type Config struct {
	Token string `envconfig:"TELEGRAM_API_TOKEN"`
	Model string `envconfig:"MODEL" default:"llama3"`
}

// Load loads the configuration from environment variables
func (c Config) Load() (Config, error) {
	cfg := c
	var err error

	// load environment variables into the Config struct
	if err = envconfig.Process("", &cfg); err != nil {
		// if there is an error, return the default config and the error
		return c, err
	}

	// return the loaded config
	return cfg, nil
}

func NewConfig() Config {
	var cfg Config
	return cfg
}

type Bot struct {
	cfg          Config
	ollamaClient *api.Client
}

func NewBot(cfg Config) (*Bot, error) {
	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	// Add a check to ensure ollamaClient is not nil
	if ollamaClient == nil {
		return nil, fmt.Errorf("Ollama client is nil after initialization")
	}

	return &Bot{
		cfg:          cfg,
		ollamaClient: ollamaClient,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	opts := []tbot.Option{
		tbot.WithDefaultHandler(b.handleText),
	}
	tg, err := tbot.New(b.cfg.Token, opts...)
	if err != nil {
		panic(err)
	}
	tg.Start(ctx)
}

// containsCodeBlock checks if text contains markdown code blocks
func containsCodeBlock(text string) bool {
	// Check for triple backtick code blocks
	return strings.Contains(text, "```")
}

func (b *Bot) handleText(
	ctx context.Context, tg *tbot.Bot, update *models.Update,
) {
	// Check if ollamaClient is nil
	if b.ollamaClient == nil {
		fmt.Println("Error: Ollama client is nil")
		tg.SendMessage(ctx, &tbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Sorry, I'm having trouble connecting to my backend. Please try again later.",
		})
		return
	}

	// Send a "typing" action to show the bot is processing
	tg.SendChatAction(
		ctx, &tbot.SendChatActionParams{
			ChatID: update.Message.Chat.ID,
			Action: models.ChatActionTyping,
		},
	)

	// Create a chat request to Ollama
	msgs := []api.Message{
		api.Message{
			Role:    "system",
			Content: "Provide very brief, concise responses",
		},
		api.Message{
			Role:    "user",
			Content: update.Message.Text,
		},
	}

	streamFalse := false
	req := &api.ChatRequest{
		Model:    b.cfg.Model,
		Messages: msgs,
		Stream:   &streamFalse,
	}

	resFn := func(resp api.ChatResponse) error {
		// Create message params
		params := &tbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   resp.Message.Content,
		}

		// If the response contains code blocks, use markdown parsing
		if containsCodeBlock(resp.Message.Content) {
			params.ParseMode = models.ParseModeMarkdown
		}

		// Send the response back to the user
		tg.SendMessage(ctx, params)
		return nil
	}

	err := b.ollamaClient.Chat(ctx, req, resFn)
	if err != nil {
		fmt.Println("Error sending message:", err)
	}
}
