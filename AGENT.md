This file provides guidance to coding agents when working with code in this repository.

## Build & Run Commands

```bash
# Run the application
go run ./cmd/main

# Build the application
go build -o banray ./cmd/main

# Run tests
go test ./...

# Run a single test
go test ./internal/bot -run TestFunctionName
```

## Environment Variables

- `TELEGRAM_API_TOKEN` - Required. Telegram bot API token
- `OPENROUTER_API_KEY` - Required. OpenRouter API key
- `OPENROUTER_BASE_URL` - OpenRouter API base URL (default: "https://openrouter.ai/api/v1")
- `OPENROUTER_MODEL` - Model to use (default: "anthropic/claude-3.5-sonnet")
- `HISTORY_LIMIT` - Max messages in conversation history (default: 10)
- `DEBUG` - Set to "true" for verbose logging with caller info

## Architecture

Banray is a Telegram bot that uses OpenRouter (via langchain-go) for AI responses. It uses [uber-go/fx](https://github.com/uber-go/fx) for dependency injection.

### Module Structure

Each package exposes an `fx.Module()` function that provides its dependencies:

- **config** - Loads environment variables via `envconfig`
- **log** - Provides `zerolog.Logger`
- **agent** - Provides `Querier` for LLM interactions via langchain-go/OpenRouter
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

### Conversation Store

`internal/agent/store.go` provides an in-memory conversation store. Uses `agent.Message` for storage. Thread-safe with `sync.RWMutex`. Keyed by conversation ID (int64).

### Bot Commands

- `/clear` - Clears conversation history for the current chat
