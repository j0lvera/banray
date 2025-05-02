package main

import (
	"github.com/j0lvera/banray/internal/ai"
	"github.com/j0lvera/banray/internal/bot"
	"github.com/j0lvera/banray/internal/config"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		config.Module(),
		ai.Module(),
		bot.Module(),
	).Run()
}
