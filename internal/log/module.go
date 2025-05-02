package log

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func Module() fx.Option {
	logWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	level := zerolog.InfoLevel
	if os.Getenv("DEBUG") == "true" {
		level = zerolog.DebugLevel
	}

	return fx.Module(
		"log",
		fx.Provide(
			zerolog.New(logWriter).
				Level(level).
				With().
				Timestamp().
				Caller().
				Logger(),
		),
	)
}
