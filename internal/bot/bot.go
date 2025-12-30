package bot

import (
	"context"
	"errors"
	"time"

	tbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/j0lvera/banray/internal/agent"
	"github.com/j0lvera/banray/internal/config"
	"github.com/j0lvera/banray/internal/db"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

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
		session, err := store.GetOrCreateSession(ctx, user.ID, cfg.SimplePrompt())
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
	session, err := store.GetOrCreateSession(ctx, user.ID, cfg.SimplePrompt())
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
		session, err = store.CreateSession(ctx, user.ID, cfg.SimplePrompt())
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
	userMessageID, err := store.AddMessage(ctx, session.ID, agent.RoleUser, update.Message.Text)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to store user message")
	}

	// 6. Send typing indicator
	tg.SendChatAction(ctx, &tbot.SendChatActionParams{
		ChatID: chatID,
		Action: models.ChatActionTyping,
	})

	// Route to agentic or simple mode
	if cfg.AgenticMode {
		handleAgenticMessage(ctx, tg, chatID, session.ID, userMessageID, update.Message.Text, querier, store, cfg, log)
	} else {
		handleSimpleMessage(ctx, tg, chatID, session.ID, userMessageID, querier, store, cfg, log)
	}
}

// handleSimpleMessage handles messages in simple (non-agentic) mode
func handleSimpleMessage(
	ctx context.Context,
	tg *tbot.Bot,
	chatID int64,
	sessionID int64,
	userMessageID int64,
	querier agent.Querier,
	store *agent.Store,
	cfg *config.Config,
	log *zerolog.Logger,
) {
	// Build messages for LLM (system prompt + history)
	messages := []agent.Message{
		{
			Role:    agent.RoleSystem,
			Content: cfg.SimplePrompt(),
		},
	}

	history, err := store.GetSessionMessages(ctx, sessionID)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to get session messages")
	}
	messages = append(messages, history...)

	// Query the LLM
	log.Info().Int64("chat_id", chatID).Int64("session_id", sessionID).Msg("ai request sending")
	result, err := querier.Query(ctx, messages)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to generate ai response")
		tg.SendMessage(ctx, &tbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Sorry, I encountered an error while processing your request.",
		})
		return
	}
	log.Info().
		Int64("chat_id", chatID).
		Int64("session_id", sessionID).
		Int("input_tokens", result.InputTokens).
		Int("output_tokens", result.OutputTokens).
		Int("total_tokens", result.TotalTokens).
		Msg("ai response received")

	// Record LLM request for usage tracking
	if err := store.RecordLLMRequest(ctx, sessionID, userMessageID, result.InputTokens, result.OutputTokens, result.TotalTokens, cfg.Model); err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to record llm request")
	}

	// Store the assistant response
	if _, err := store.AddMessage(ctx, sessionID, agent.RoleAssistant, result.Content); err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to store bot message")
	}

	// Send response to user
	tg.SendMessage(ctx, &tbot.SendMessageParams{
		ChatID: chatID,
		Text:   result.Content,
	})
}

// handleAgenticMessage handles messages in agentic mode with bash access
func handleAgenticMessage(
	ctx context.Context,
	tg *tbot.Bot,
	chatID int64,
	sessionID int64,
	userMessageID int64,
	userText string,
	querier agent.Querier,
	store *agent.Store,
	cfg *config.Config,
	log *zerolog.Logger,
) {
	// Load conversation history (excluding the current message we just stored)
	allMessages, err := store.GetSessionMessages(ctx, sessionID)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to get session messages")
	}

	// Exclude the last message (the one we just stored) since we pass userText separately
	var history []agent.Message
	if len(allMessages) > 1 {
		history = allMessages[:len(allMessages)-1]
	}

	// Create runner config
	runnerConfig := agent.RunnerConfig{
		MaxSteps:         cfg.MaxSteps,
		CommandTimeout:   cfg.CommandTimeout,
		WorkingDir:       cfg.WorkingDir,
		SystemPrompt:     cfg.AgentPrompt(),
		ContextThreshold: cfg.ContextThreshold,
	}

	// Create the runner
	runner := agent.NewRunner(runnerConfig, querier, log)

	// Start a goroutine to send typing indicators periodically
	typingCtx, cancelTyping := context.WithCancel(ctx)
	defer cancelTyping()
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				tg.SendChatAction(typingCtx, &tbot.SendChatActionParams{
					ChatID: chatID,
					Action: models.ChatActionTyping,
				})
			}
		}
	}()

	// Run the agentic loop
	log.Info().
		Int64("chat_id", chatID).
		Int64("session_id", sessionID).
		Bool("agentic_mode", true).
		Int("max_steps", cfg.MaxSteps).
		Int("history_messages", len(history)).
		Msg("starting agentic run")

	result, err := runner.Run(ctx, history, userText)
	if err != nil {
		// Check if it's a step limit termination
		var termErr *agent.TerminatingErr
		if errors.As(err, &termErr) && termErr.Reason == agent.ReasonStepLimit {
			log.Warn().
				Int64("chat_id", chatID).
				Int("steps", result.Steps).
				Msg("agentic run hit step limit")
		} else {
			log.Error().Err(err).Int64("chat_id", chatID).Msg("agentic run failed")
			tg.SendMessage(ctx, &tbot.SendMessageParams{
				ChatID: chatID,
				Text:   "Sorry, I encountered an error while processing your request.",
			})
			return
		}
	}

	log.Info().
		Int64("chat_id", chatID).
		Int64("session_id", sessionID).
		Int("steps", result.Steps).
		Int("input_tokens", result.TokenUsage.InputTokens).
		Int("output_tokens", result.TokenUsage.OutputTokens).
		Int("total_tokens", result.TokenUsage.TotalTokens).
		Str("terminated_by", string(result.Reason)).
		Msg("agentic run complete")

	// Record LLM request with aggregated tokens
	if err := store.RecordLLMRequest(ctx, sessionID, userMessageID, result.TokenUsage.InputTokens, result.TokenUsage.OutputTokens, result.TokenUsage.TotalTokens, cfg.Model); err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to record llm request")
	}

	// Store the assistant response
	if _, err := store.AddMessage(ctx, sessionID, agent.RoleAssistant, result.Response); err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("unable to store bot message")
	}

	// Send response to user
	tg.SendMessage(ctx, &tbot.SendMessageParams{
		ChatID: chatID,
		Text:   result.Response,
	})
}
