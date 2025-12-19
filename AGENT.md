This file provides guidance to coding agnets when working with code in this repository.

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
- **agent** - Provides `bot.Service` implementation using langchain-go/OpenRouter
- **bot** - Telegram bot with message handling, starts via fx lifecycle hook

### Key Interfaces

The `bot.Service` interface (in `internal/bot/types.go`) defines the AI backend contract:
```go
type Service interface {
    Reply(ctx context.Context, prompt string, history []Message) (Response, error)
}
```

The `agent.Service` implements this interface using langchain-go's OpenAI-compatible client with OpenRouter.

### Conversation Store

`internal/bot/store.go` provides an in-memory conversation store keyed by Telegram chat ID. Thread-safe with `sync.RWMutex`.

### Bot Commands

- `/clear` - Clears conversation history for the current chat
