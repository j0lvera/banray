package agent

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/j0lvera/banray/internal/db"
	dbgen "github.com/j0lvera/banray/internal/db/gen"
)

// Store manages conversation sessions and messages using PostgreSQL
type Store struct {
	client *db.Client
}

// NewStore creates a new conversation store
func NewStore(client *db.Client) *Store {
	return &Store{client: client}
}

// GetOrCreateSession returns the active session for a user, creating one if none exists
func (s *Store) GetOrCreateSession(ctx context.Context, userID int64, systemPrompt string) (dbgen.DataSession, error) {
	session, err := s.client.Queries.GetActiveSession(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No active session, create one
			return s.client.Queries.CreateSession(ctx, dbgen.CreateSessionParams{
				UserID:       userID,
				SystemPrompt: pgtype.Text{String: systemPrompt, Valid: systemPrompt != ""},
			})
		}
		return dbgen.DataSession{}, err
	}
	return session, nil
}

// CreateSession creates a new session for a user
func (s *Store) CreateSession(ctx context.Context, userID int64, systemPrompt string) (dbgen.DataSession, error) {
	return s.client.Queries.CreateSession(ctx, dbgen.CreateSessionParams{
		UserID:       userID,
		SystemPrompt: pgtype.Text{String: systemPrompt, Valid: systemPrompt != ""},
	})
}

// EndSession marks a session as ended
func (s *Store) EndSession(ctx context.Context, sessionID int64) error {
	return s.client.Queries.EndSession(ctx, sessionID)
}

// AddMessage adds a message to a session
func (s *Store) AddMessage(ctx context.Context, sessionID int64, role Role, content string) error {
	return s.client.Queries.AddMessage(ctx, dbgen.AddMessageParams{
		SessionID: sessionID,
		Role:      string(role),
		Content:   content,
	})
}

// GetSessionMessages returns all messages in a session
func (s *Store) GetSessionMessages(ctx context.Context, sessionID int64) ([]Message, error) {
	rows, err := s.client.Queries.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, len(rows))
	for i, row := range rows {
		messages[i] = Message{
			Role:    Role(row.Role),
			Content: row.Content,
		}
	}
	return messages, nil
}

// CountSessionMessages returns the number of messages in a session
func (s *Store) CountSessionMessages(ctx context.Context, sessionID int64) (int, error) {
	count, err := s.client.Queries.CountSessionMessages(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
