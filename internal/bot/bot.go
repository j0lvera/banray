package bot

import (
	"context"
	"fmt"
	"strings"

	tbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/j0lvera/banray/internal/config"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config  *config.Config
	Service Service
}

type Result struct {
	fx.Out

	Bot   *tbot.Bot
	Store *Store
}

func New(lc fx.Lifecycle, p Params) (Result, error) {
	// Create conversation store
	store := NewStore()

	opts := []tbot.Option{
		tbot.WithDefaultHandler(
			func(ctx context.Context, tg *tbot.Bot, update *models.Update) {
				handleMessage(ctx, tg, update, p.Service, store, p.Config)
			},
		),
	}

	tg, err := tbot.New(p.Config.Token, opts...)
	if err != nil {
		return Result{}, err
	}

	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				fmt.Println("Starting Telegram bot...")
				go tg.Start(context.Background())
				return nil
			},
			OnStop: func(ctx context.Context) error {
				fmt.Println("Stopping Telegram bot...")
				return nil
			},
		},
	)

	return Result{
		Bot:   tg,
		Store: store,
	}, nil
}

func Module() fx.Option {
	return fx.Module(
		"bot",
		fx.Provide(
			New,
		),
		fx.Invoke(
			func(bot *tbot.Bot) {},
		),
	)
}

func handleMessage(
	ctx context.Context,
	tg *tbot.Bot,
	update *models.Update,
	svc Service,
	store *Store,
	config *config.Config,
) {
	// Check for command to clear history
	if update.Message.Text == "/clear" {
		store.Clear(update.Message.Chat.ID)
		tg.SendMessage(
			ctx, &tbot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Conversation history cleared.",
			},
		)
		return
	}

	// Check if we need to clear history due to exceeding the limit
	// We check before adding the new message, and we account for the fact
	// that we'll be adding both a user message and a bot response (2 messages)
	if store.Length(update.Message.Chat.ID) >= config.HistoryLimit-1 {
		store.Clear(update.Message.Chat.ID)
		// Optionally notify the user that history was cleared
		tg.SendMessage(
			ctx, &tbot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Conversation history was automatically cleared due to length.",
			},
		)
	}

	// Store the user message
	store.AddUserMessage(update.Message.Chat.ID, update.Message.Text)

	// Send a "typing" action to show the bot is processing
	tg.SendChatAction(
		ctx, &tbot.SendChatActionParams{
			ChatID: update.Message.Chat.ID,
			Action: models.ChatActionTyping,
		},
	)

	// Get conversation history
	history := store.History(
		update.Message.Chat.ID, config.HistoryLimit,
	)

	// Generate response using the AI service
	response, err := svc.Reply(ctx, update.Message.Text, history)
	if err != nil {
		fmt.Println("Error generating response:", err)
		tg.SendMessage(
			ctx, &tbot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Sorry, I encountered an error while processing your request.",
			},
		)
		return
	}

	// Store the bot response
	store.AddBotMessage(update.Message.Chat.ID, response.Content)

	// Create message params
	params := &tbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
	}

	// If the response contains markdown elements, enable markdown parsing
	if containsCodeBlock(response.Content) {
		params.ParseMode = models.ParseModeMarkdown
		params.Text = response.Content
	} else {
		params.Text = response.Content
	}

	// Send the response back to the user
	tg.SendMessage(ctx, params)
}

// containsCodeBlock checks if text contains any markdown elements
func containsCodeBlock(text string) bool {
	// Check for any markdown elements
	return strings.Contains(text, "```")
}
