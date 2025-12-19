package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/j0lvera/banray/internal/config"
	"github.com/j0lvera/banray/internal/db/gen"
)

type Params struct {
	fx.In

	Config *config.Config
	Logger zerolog.Logger
}

type Result struct {
	fx.Out

	Client *Client
}

type Client struct {
	Queries *dbgen.Queries
	Pool    *pgxpool.Pool
}

func NewClient(pool *pgxpool.Pool) *Client {
	return &Client{
		Queries: dbgen.New(pool),
		Pool:    pool,
	}
}

func New(lc fx.Lifecycle, p Params) (Result, error) {
	pool, err := pgxpool.New(context.Background(), p.Config.DatabaseURL)
	if err != nil {
		return Result{}, fmt.Errorf("unable to create connection pool: %w", err)
	}

	client := NewClient(pool)

	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				p.Logger.Info().Msg("database connection established")
				return pool.Ping(ctx)
			},
			OnStop: func(ctx context.Context) error {
				p.Logger.Info().Msg("closing database connection")
				pool.Close()
				return nil
			},
		},
	)

	return Result{Client: client}, nil
}

func Module() fx.Option {
	return fx.Module(
		"db",
		fx.Provide(New),
	)
}
