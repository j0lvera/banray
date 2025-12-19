package bot

import (
	"context"

	tbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/j0lvera/banray/internal/agent"
	"github.com/j0lvera/banray/internal/config"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config  *config.Config
	Querier agent.Querier
}

type Result struct {
	fx.Out

	Bot   *tbot.Bot
	Store *agent.Store
}

func New(lc fx.Lifecycle, p Params, log zerolog.Logger) (Result, error) {
	store := agent.NewStore()

	opts := []tbot.Option{
		tbot.WithDefaultHandler(
			func(ctx context.Context, tg *tbot.Bot, update *models.Update) {
				handleMessage(ctx, tg, update, p.Querier, store, p.Config, &log)
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
	querier agent.Querier,
	store *agent.Store,
	cfg *config.Config,
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
	if store.Length(chatID) >= cfg.HistoryLimit-1 {
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
	store.AddMessage(chatID, agent.RoleUser, update.Message.Text)

	// Send a "typing" action to show the bot is processing
	tg.SendChatAction(
		ctx, &tbot.SendChatActionParams{
			ChatID: chatID,
			Action: models.ChatActionTyping,
		},
	)

	// Build messages for the LLM
	messages := []agent.Message{
		{
			Role:    agent.RoleSystem,
			Content: "Provide brief, concise responses with a friendly and human tone.",
		},
	}

	// Add conversation history
	history := store.History(chatID, cfg.HistoryLimit)
	messages = append(messages, history...)

	// Query the LLM
	log.Info().Int64("chat_id", chatID).Msg("ai request sending")
	response, err := querier.Query(ctx, messages)
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
	store.AddMessage(chatID, agent.RoleAssistant, response)

	// Send the response back to the user
	tg.SendMessage(
		ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   response,
		},
	)
}
