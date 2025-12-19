package ai

import (
	"github.com/j0lvera/banray/internal/bot"
	"github.com/j0lvera/banray/internal/config"
	"go.uber.org/fx"
)

// Params for creating an AI service
type Params struct {
	fx.In

	Config *config.Config
}

// Result of creating an AI service
type Result struct {
	fx.Out

	Service bot.Service
}

// New creates a new AI service based on configuration
func New(p Params) (Result, error) {
	service, err := NewService(p.Config.APIKey, p.Config.BaseURL, p.Config.Model)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Service: service,
	}, nil
}

// Module provides the AI service
func Module() fx.Option {
	return fx.Module(
		"ai",
		fx.Provide(
			New,
		),
	)
}
