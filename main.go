package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg := NewConfig()
	cfg, err := cfg.Load()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	bot, err := NewBot(cfg)
	if err != nil {
		fmt.Println("Error creating bot:", err)
		return
	}
	bot.Start(ctx)
}
