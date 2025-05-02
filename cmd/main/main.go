package main

import (
	"os"
	"time"

	"github.com/ipfans/fxlogger"
	"github.com/j0lvera/banray/internal/ai"
	"github.com/j0lvera/banray/internal/bot"
	"github.com/j0lvera/banray/internal/config"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func main() {
	logWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	level := zerolog.InfoLevel
	if os.Getenv("DEBUG") == "true" {
		level = zerolog.DebugLevel
	}

	fx.New(
		config.Module(),
		ai.Module(),
		bot.Module(),
		fx.WithLogger(
			fxlogger.WithZerolog(
				zerolog.New(logWriter).
					Level(level).
					With().
					Timestamp().
					Caller().
					Logger(),
			),
		),
	).Run()
}
