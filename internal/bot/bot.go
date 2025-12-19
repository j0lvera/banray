package bot

import (
	"context"

	tbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/j0lvera/banray/internal/agent"
	"github.com/j0lvera/banray/internal/config"
	"github.com/j0lvera/banray/internal/db"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const defaultSystemPrompt = "Provide brief, concise responses with a friendly and human tone. Do not use markdown formatting."

type Params struct {
	fx.In

	Config   *config.Config
	Querier  agent.Querier
	DBClient *db.Client
}

type Result struct {
	fx.Out

	Bot       *tbot.Bot
	Store     *agent.Store
	UserStore *agent.UserStore
}

func New(lc fx.Lifecycle, p Params, log zerolog.Logger) (Result, error) {
	store := agent.NewStore(p.DBClient)
	userStore := agent.NewUserStore(p.DBClient)

	opts := []tbot.Option{
		tbot.WithDefaultHandler(
			func(ctx context.Context, tg *tbot.Bot, update *models.Update) {
				handleMessage(ctx, tg, update, p.Querier, store, userStore, p.Config, &log)
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
		Bot:       tg,
		Store:     store,
		UserStore: userStore,
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
	userStore *agent.UserStore,
	cfg *config.Config,
	log *zerolog.Logger,
) {
	// Guard against nil message
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID

	// Guard against nil user
	if update.Message.From == nil {
		log.Warn().Int64("chat_id", chatID).Msg("received message without user info")
		return
	}

	// 1. Upsert user from Telegram data
	user, err := userStore.UpsertUser(
		ctx,
		update.Message.From.ID,
		update.Message.From.Username,
		update.Message.From.FirstName,
		update.Message.From.LastName,
		update.Message.From.LanguageCode,
	)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to upsert user")
		tg.SendMessage(ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Sorry, I encountered an error. Please try again.",
		})
		return
	}

	// 2. Handle /clear command
	if update.Message.Text == "/clear" {
		// Get active session to end it
		session, err := store.GetOrCreateSession(ctx, user.ID, defaultSystemPrompt)
		if err != nil {
			log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to get session for clear")
		} else {
			if err := store.EndSession(ctx, session.ID); err != nil {
				log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to end session")
			}
		}
		tg.SendMessage(ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Conversation cleared. Starting fresh!",
		})
		log.Info().Int64("chat_id", chatID).Int64("user_id", user.ID).Msg("session ended by user")
		return
	}

	// 3. Get or create active session
	session, err := store.GetOrCreateSession(ctx, user.ID, defaultSystemPrompt)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to get or create session")
		tg.SendMessage(ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Sorry, I encountered an error. Please try again.",
		})
		return
	}

	// 4. Check if session hit message limit
	count, err := store.CountSessionMessages(ctx, session.ID)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to count session messages")
	}
	if count >= cfg.HistoryLimit {
		// End current session and create new one
		if err := store.EndSession(ctx, session.ID); err != nil {
			log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to end session at limit")
		}
		session, err = store.CreateSession(ctx, user.ID, defaultSystemPrompt)
		if err != nil {
			log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to create new session")
			tg.SendMessage(ctx, &tbot.SendMessageParams{
				ChatID: chatID,
				Text:   "Sorry, I encountered an error. Please try again.",
			})
			return
		}
		tg.SendMessage(ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Starting a new conversation due to context limit.",
		})
		log.Info().Int64("chat_id", chatID).Int64("user_id", user.ID).Msg("session auto-rotated due to limit")
	}

	// 5. Store the user message
	if err := store.AddMessage(ctx, session.ID, agent.RoleUser, update.Message.Text); err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to store user message")
	}

	// 6. Send typing indicator
	tg.SendChatAction(ctx, &tbot.SendChatActionParams{
		ChatID: chatID,
		Action: models.ChatActionTyping,
	})

	// 7. Build messages for LLM (system prompt + history)
	messages := []agent.Message{
		{
			Role:    agent.RoleSystem,
			Content: defaultSystemPrompt,
		},
	}

	history, err := store.GetSessionMessages(ctx, session.ID)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to get session messages")
	}
	messages = append(messages, history...)

	// 8. Query the LLM
	log.Info().Int64("chat_id", chatID).Int64("session_id", session.ID).Msg("ai request sending")
	response, err := querier.Query(ctx, messages)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to generate ai response")
		tg.SendMessage(ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Sorry, I encountered an error while processing your request.",
		})
		return
	}
	log.Info().Int64("chat_id", chatID).Int64("session_id", session.ID).Msg("ai response received")

	// 9. Store the assistant response
	if err := store.AddMessage(ctx, session.ID, agent.RoleAssistant, response); err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to store bot message")
	}

	// 10. Send response to user
	tg.SendMessage(ctx, &tbot.SendMessageParams{
		ChatID: chatID,
		Text:   response,
	})
}
