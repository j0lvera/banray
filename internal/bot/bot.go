package bot

import (
	"context"
	"fmt"

	tbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/j0lvera/banray/internal/config"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config *config.Config
}

type Result struct {
	fx.Out

	Bot *tbot.Bot
}

func New(lc fx.Lifecycle, p Params) (Result, error) {
	opts := []tbot.Option{
		tbot.WithDefaultHandler(handler),
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
		Bot: tg,
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

func handler(ctx context.Context, tg *tbot.Bot, update *models.Update) {
	tg.SendMessage(
		ctx, &tbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   update.Message.Text,
		},
	)
}
