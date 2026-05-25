package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"

	"github.com/go-telegram/bot"
)

func main() {
	_ = godotenv.Load()

	cfg, cfgError := loadConfig()
	if cfgError != nil {
		log.Fatalf("Failed to load the configuration with error %v", cfgError)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	store, err := openUserStore(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open the database: %v", err)
	}
	defer store.close()

	app := newBotApp(store, newPredictClient(cfg.APIURL))

	b, err := bot.New(cfg.BotToken, bot.WithDefaultHandler(app.handleUpdate))
	if err != nil {
		log.Fatalf("Failed to start bot %v", err)
	}

	b.Start(ctx)
}
