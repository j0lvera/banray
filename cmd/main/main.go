package main

import (
	"github.com/ipfans/fxlogger"
	"github.com/j0lvera/banray/internal/agent"
	"github.com/j0lvera/banray/internal/bot"
	"github.com/j0lvera/banray/internal/config"
	"github.com/j0lvera/banray/internal/db"
	"github.com/j0lvera/banray/internal/log"
	"go.uber.org/fx"
)

func main() {
	// Create the logger first
	logger := log.NewFxLogger()

	fx.New(
		config.Module(),
		log.Module(),
		db.Module(),
		agent.Module(),
		bot.Module(),
		// Use the same logger for fx
		fx.WithLogger(
			fxlogger.WithZerolog(logger),
		),
	).Run()
}
