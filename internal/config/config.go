package config

import (
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/fx"
)

type Config struct {
	Token string `envconfig:"TELEGRAM_API_TOKEN"`
}

// LoadEnv loads the configuration from environment variables
func (c Config) LoadEnv() (Config, error) {
	cfg := c

	// load environment variables into the Config struct
	if err := envconfig.Process("", &cfg); err != nil {
		// if there is an error, return the default config and the error
		return c, err
	}

	// return the loaded config
	return cfg, nil
}

func NewConfig() (*Config, error) {
	var cfg Config
	loadedCfg, err := cfg.LoadEnv()
	if err != nil {
		return nil, err
	}
	return &loadedCfg, nil
}

func Module() fx.Option {
	return fx.Module(
		"config",
		fx.Provide(
			NewConfig,
		),
	)
}
