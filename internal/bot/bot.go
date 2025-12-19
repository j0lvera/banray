package bot

import (
	"context"

	tbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/j0lvera/banray/internal/config"
	"github.com/rs/zerolog"
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

func New(lc fx.Lifecycle, p Params, log zerolog.Logger) (Result, error) {
	// Create conversation store
	store := NewStore()

	opts := []tbot.Option{
		tbot.WithDefaultHandler(
			func(ctx context.Context, tg *tbot.Bot, update *models.Update) {
				handleMessage(ctx, tg, update, p.Service, store, p.Config, &log)
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
				log.Info().Msg("starting telegram bot...")
				go tg.Start(context.Background())
				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Info().Msg("stopping telegram bot...")
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
	log *zerolog.Logger,
) {
	chatID := update.Message.Chat.ID

	// Check for command to clear history
	if update.Message.Text == "/clear" {
		store.Clear(chatID)
		tg.SendMessage(
			ctx, &tbot.SendMessageParams{
				ChatID: chatID,
				Text:   "Conversation history cleared.",
			},
		)
		log.Info().Int64("chat_id", chatID).Msg("history cleared")
		return
	}

	// Check if we need to clear history due to exceeding the limit
	if store.Length(chatID) >= config.HistoryLimit-1 {
		store.Clear(chatID)
		tg.SendMessage(
			ctx, &tbot.SendMessageParams{
				ChatID: chatID,
				Text:   "Conversation history was automatically cleared due to length.",
			},
		)
		log.Info().Int64("chat_id", chatID).Msg("history auto-cleared")
	}

	// Store the user message
	store.AddUserMessage(chatID, update.Message.Text)

	// Send a "typing" action to show the bot is processing
	tg.SendChatAction(
		ctx, &tbot.SendChatActionParams{
			ChatID: chatID,
			Action: models.ChatActionTyping,
		},
	)

	// Get conversation history
	history := store.History(chatID, config.HistoryLimit)

	// Generate response using the AI service
	log.Info().Int64("chat_id", chatID).Msg("ai request sending")
	response, err := svc.Reply(ctx, update.Message.Text, history)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to generate ai response")
		tg.SendMessage(
			ctx, &tbot.SendMessageParams{
				ChatID: chatID,
				Text:   "Sorry, I encountered an error while processing your request.",
			},
		)
		return
	}
	log.Info().Int64("chat_id", chatID).Msg("ai response received")

	// Store the bot response
	store.AddBotMessage(chatID, response.Content)

	// Send the response back to the user
	tg.SendMessage(
		ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   response.Content,
		},
	)
}
