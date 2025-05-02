package log

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// NewLogger creates a configured zerolog.Logger instance
func NewLogger() zerolog.Logger {

	level := zerolog.InfoLevel
	if os.Getenv("DEBUG") == "true" {
		level = zerolog.DebugLevel

		logWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}

		return zerolog.New(logWriter).
			Level(level).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	return zerolog.New(os.Stdout)
}

// NewFxLogger creates a configured zerolog.Logger instance
func NewFxLogger() zerolog.Logger {
	// i know it's duplicated code, but i don't want to find a clever
	// way to pass the custom logger to the fx module.
	// the only difference is that we aren't using the caller.
	level := zerolog.InfoLevel
	if os.Getenv("DEBUG") == "true" {
		level = zerolog.DebugLevel

		logWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}

		return zerolog.New(logWriter).
			Level(level).
			With().
			Timestamp().
			Logger()
	}

	return zerolog.New(os.Stdout)
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
