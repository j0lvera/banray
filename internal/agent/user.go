package agent

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/j0lvera/banray/internal/db"
	dbgen "github.com/j0lvera/banray/internal/db/gen"
)

// UserStore manages user data using PostgreSQL
type UserStore struct {
	client *db.Client
}

// NewUserStore creates a new user store
func NewUserStore(client *db.Client) *UserStore {
	return &UserStore{client: client}
}

// UpsertUser creates or updates a user from Telegram data
func (s *UserStore) UpsertUser(ctx context.Context, telegramID int64, username, firstName, lastName, languageCode string) (*dbgen.DataUser, error) {
	return s.client.Queries.UpsertUser(ctx, dbgen.UpsertUserParams{
		TelegramID:   telegramID,
		Username:     pgtype.Text{String: username, Valid: username != ""},
		FirstName:    pgtype.Text{String: firstName, Valid: firstName != ""},
		LastName:     pgtype.Text{String: lastName, Valid: lastName != ""},
		LanguageCode: pgtype.Text{String: languageCode, Valid: languageCode != ""},
	})
}

// GetUserByTelegramID retrieves a user by their Telegram ID
func (s *UserStore) GetUserByTelegramID(ctx context.Context, telegramID int64) (*dbgen.DataUser, error) {
	return s.client.Queries.GetUserByTelegramID(ctx, telegramID)
}
