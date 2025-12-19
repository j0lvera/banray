package agent

import (
	"github.com/j0lvera/banray/internal/config"
	"go.uber.org/fx"
)

// Params for creating a Querier
type Params struct {
	fx.In

	Config *config.Config
}

// Result of creating a Querier
type Result struct {
	fx.Out

	Querier Querier
}

// New creates a new Querier based on configuration
func New(p Params) (Result, error) {
	querier, err := NewOpenAIQuerier(p.Config.APIKey, p.Config.BaseURL, p.Config.Model)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Querier: querier,
	}, nil
}

// Module provides the agent Querier
func Module() fx.Option {
	return fx.Module(
		"agent",
		fx.Provide(
			New,
		),
	)
}
