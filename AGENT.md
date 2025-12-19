This file provides guidance to coding agents when working with code in this repository.

## Build & Run Commands

```bash
# Run the application
go run ./cmd/main

# Build the application
go build -o bin/banray ./cmd/main

# Run tests
go test ./...

# Run a single test
go test ./internal/bot -run TestFunctionName

# Generate sqlc code
sqlc generate

# Run migrations (using goose)
goose -dir internal/db/migrations postgres "$DATABASE_URL" up
```

## Environment Variables

- `TELEGRAM_API_TOKEN` - Required. Telegram bot API token
- `OPENROUTER_API_KEY` - Required. OpenRouter API key
- `OPENROUTER_BASE_URL` - OpenRouter API base URL (default: "https://openrouter.ai/api/v1")
- `OPENROUTER_MODEL` - Model to use (default: "anthropic/claude-3.5-sonnet")
- `DATABASE_URL` - Required. PostgreSQL connection string
- `HISTORY_LIMIT` - Max messages per session before auto-rotation (default: 10)
- `DEBUG` - Set to "true" for verbose logging with caller info

## Architecture

Banray is a Telegram bot that uses OpenRouter (via langchain-go) for AI responses. It uses [uber-go/fx](https://github.com/uber-go/fx) for dependency injection.

### Module Structure

Each package exposes an `fx.Module()` function that provides its dependencies:

- **config** - Loads environment variables via `envconfig`
- **log** - Provides `zerolog.Logger`
- **db** - Provides `*db.Client` with pgxpool connection and sqlc queries
- **agent** - Provides `Querier` for LLM interactions, `Store` for sessions/messages, `UserStore` for user data
- **bot** - Telegram bot with message handling, starts via fx lifecycle hook

### Key Types

The `agent` package defines:
```go
type Role string  // RoleSystem, RoleUser, RoleAssistant

type Message struct {
    Role    Role
    Content string
}

type Querier interface {
    Query(ctx context.Context, messages []Message) (string, error)
}
```

`OpenAIQuerier` implements `Querier` using langchain-go's OpenAI-compatible client with OpenRouter.

### Database Schema

Uses PostgreSQL with sqlc for type-safe queries. Entity hierarchy:

```
users (Telegram user info)
└── sessions (bounded context windows)
    └── messages (conversation content)
```

**Tables:**

- `data.users` - Telegram user info (telegram_id, username, first_name, last_name, language_code)
- `data.sessions` - Context windows per user. Ended when limit reached or `/clear` called.
- `data.messages` - Messages within a session (role, content)

**Key concept:** Sessions are bounded context windows. When `HISTORY_LIMIT` is reached or user sends `/clear`, the current session ends and a new one starts. History is preserved (not deleted).

**Files:**

- Migrations in `internal/db/migrations/`
- Queries in `internal/db/queries/`
- Generated code in `internal/db/gen/`
- `utils.nanoid()` function for URL-friendly unique IDs

### Stores

- `agent.Store` - Manages sessions and messages
  - `GetOrCreateSession(ctx, userID, systemPrompt)` - Get active session or create new
  - `EndSession(ctx, sessionID)` - Mark session as ended
  - `AddMessage(ctx, sessionID, role, content)` - Add message to session
  - `GetSessionMessages(ctx, sessionID)` - Get all messages in session
  - `CountSessionMessages(ctx, sessionID)` - Count messages in session

- `agent.UserStore` - Manages user data
  - `UpsertUser(ctx, telegramID, username, firstName, lastName, languageCode)` - Create or update user

### Bot Commands

- `/clear` - Ends current session, next message starts fresh context

### Message Flow

1. Upsert user from Telegram update
2. Get or create active session for user
3. Check if session hit message limit → auto-rotate if needed
4. Store user message
5. Build LLM context (system prompt + session history)
6. Query LLM
7. Store assistant response
8. Send response to user
