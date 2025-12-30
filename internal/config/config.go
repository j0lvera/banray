package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/fx"
)

// Config holds all configuration from environment variables.
type Config struct {
	Token        string `envconfig:"TELEGRAM_API_TOKEN" required:"true"`
	APIKey       string `envconfig:"OPENROUTER_API_KEY" required:"true"`
	BaseURL      string `envconfig:"OPENROUTER_BASE_URL" default:"https://openrouter.ai/api/v1"`
	Model        string `envconfig:"OPENROUTER_MODEL" default:"anthropic/claude-3.5-sonnet"`
	HistoryLimit int    `envconfig:"HISTORY_LIMIT" default:"10"`
	DatabaseURL  string `envconfig:"DATABASE_URL" required:"true"`

	// Agentic mode settings
	AgenticMode      bool          `envconfig:"AGENTIC_MODE" default:"false"`
	MaxSteps         int           `envconfig:"MAX_STEPS" default:"10"`
	CommandTimeout   time.Duration `envconfig:"COMMAND_TIMEOUT" default:"30s"`
	WorkingDir       string        `envconfig:"WORKING_DIR" default:""`
	ContextThreshold int           `envconfig:"CONTEXT_THRESHOLD" default:"8000"` // Chars before summarization kicks in

	// Path to config.toml file
	ConfigFile string `envconfig:"CONFIG_FILE" default:"config.toml"`

	// Path to context directory containing .md files to inject into prompts
	ContextDir string `envconfig:"CONTEXT_DIR" default:".context"`

	// Prompts loaded from config.toml
	Prompts Prompts

	// Context loaded from .context/*.md files
	Context string
}

// Prompts holds system prompts loaded from config.toml.
type Prompts struct {
	Simple string `toml:"simple"`
	Agent  string `toml:"agent"`
}

// FileConfig represents the structure of config.toml.
type FileConfig struct {
	Prompts Prompts `toml:"prompts"`
}

// DefaultPrompts provides fallback prompts if config.toml is not found.
var DefaultPrompts = Prompts{
	Simple: "Provide brief, concise responses with a friendly and human tone. Do not use markdown formatting.",
	Agent: `You are an autonomous agent with bash access. You can execute commands to accomplish tasks.

RULES:
1. Respond with exactly ONE bash command in a code block like this:
` + "```bash" + `
your command here
` + "```" + `

2. After each command, you'll see the output. Use it to decide your next action.

3. When the task is complete, run:
` + "```bash" + `
echo "TASK_COMPLETE"
echo "Your final summary here"
` + "```" + `

4. Be concise. Execute commands, observe results, iterate.

5. If a command fails, try an alternative approach.

6. You have access to common tools: curl, jq, python3, node, etc.`,
}

// LoadEnv loads the configuration from environment variables.
func (c Config) LoadEnv() (Config, error) {
	cfg := c

	if err := envconfig.Process("", &cfg); err != nil {
		return c, err
	}

	return cfg, nil
}

// LoadFile loads prompts from config.toml file.
func (c *Config) LoadFile() error {
	// Try to find config file
	configPath := c.ConfigFile
	if !filepath.IsAbs(configPath) {
		// Try current directory first
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Try executable directory
			execPath, err := os.Executable()
			if err == nil {
				execDir := filepath.Dir(execPath)
				configPath = filepath.Join(execDir, c.ConfigFile)
			}
		}
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Use defaults if no config file
		c.Prompts = DefaultPrompts
		return nil
	}

	// Load TOML file
	var fileConfig FileConfig
	if _, err := toml.DecodeFile(configPath, &fileConfig); err != nil {
		return err
	}

	c.Prompts = fileConfig.Prompts

	// Use defaults for empty prompts
	if c.Prompts.Simple == "" {
		c.Prompts.Simple = DefaultPrompts.Simple
	}
	if c.Prompts.Agent == "" {
		c.Prompts.Agent = DefaultPrompts.Agent
	}

	return nil
}

// LoadContext loads all .md files from the context directory and concatenates them.
func (c *Config) LoadContext() error {
	// Check if context directory exists
	if _, err := os.Stat(c.ContextDir); os.IsNotExist(err) {
		// No context directory, that's fine
		return nil
	}

	// Find all .md files in context directory
	pattern := filepath.Join(c.ContextDir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob context files: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	// Read and concatenate all files
	var parts []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read context file %s: %w", file, err)
		}
		parts = append(parts, string(content))
	}

	c.Context = strings.Join(parts, "\n\n---\n\n")

	return nil
}

// AgentPrompt returns the agent prompt with context injected.
func (c *Config) AgentPrompt() string {
	// Prepend today's date so the agent knows the current date
	dateHeader := fmt.Sprintf("Today's date: %s\n\n", time.Now().Format("2006-01-02"))

	if c.Context == "" {
		return dateHeader + c.Prompts.Agent
	}
	return dateHeader + c.Prompts.Agent + "\n\n## Available Tools & Context\n\n" + c.Context
}

// SimplePrompt returns the simple prompt with context injected.
func (c *Config) SimplePrompt() string {
	if c.Context == "" {
		return c.Prompts.Simple
	}
	return c.Prompts.Simple + "\n\n## Available Tools & Context\n\n" + c.Context
}

func NewConfig() (*Config, error) {
	var cfg Config
	loadedCfg, err := cfg.LoadEnv()
	if err != nil {
		return nil, err
	}

	// Load prompts from config.toml
	if err := loadedCfg.LoadFile(); err != nil {
		return nil, err
	}

	// Load context from .context/*.md files
	if err := loadedCfg.LoadContext(); err != nil {
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
