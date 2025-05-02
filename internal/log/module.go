package log

import (
	"os"
	"time"

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

// Module provides an fx.Option that creates a logger
func Module() fx.Option {
	return fx.Module(
		"log",
		fx.Provide(
			NewLogger,
		),
	)
}
