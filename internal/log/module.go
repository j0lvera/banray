package log

import (
	"os"
	"time"

	"github.com/ipfans/fxlogger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// NewLogger creates a configured zerolog.Logger instance
func NewLogger() zerolog.Logger {
	logWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	level := zerolog.InfoLevel
	if os.Getenv("DEBUG") == "true" {
		level = zerolog.DebugLevel
	}

	return zerolog.New(logWriter).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()
}

func Module() fx.Option {
	return fx.Module(
		"log",
		fx.Provide(
			NewLogger,
		),
		// Use fxlogger to provide a zerolog adapter for Fx's logging
		fx.WithLogger(func(log zerolog.Logger) fx.Logger {
			return fxlogger.WithZerolog(log)
		}),
	)
}
