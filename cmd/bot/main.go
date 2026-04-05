package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/invoice-bot/internal/bot"
	"github.com/invoice-bot/internal/config"
	"github.com/invoice-bot/internal/invoice"
	"github.com/invoice-bot/internal/llm"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded")

	// Load style
	style, err := config.LoadStyle(cfg.InvoiceStylePath)
	if err != nil {
		log.Fatalf("Failed to load style: %v", err)
	}
	log.Println("Style loaded")

	// Initialize Telegram bot
	tgBot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}
	log.Printf("Authorized on Telegram as @%s", tgBot.Self.UserName)

	// Initialize components
	auth := bot.NewAuth(cfg.AllowedUserIDs)
	parser := llm.NewParser(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	pdfGen := invoice.NewPDFGenerator(style)

	if len(cfg.AllowedUserIDs) > 0 {
		log.Printf("User allowlist enabled: %d user(s)", len(cfg.AllowedUserIDs))
	} else {
		log.Println("User allowlist disabled: all users can access the bot")
	}

	// Create handler
	handler := bot.NewHandler(tgBot, auth, parser, pdfGen, cfg)

	// Context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Run the bot
	if err := handler.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Bot error: %v", err)
	}

	log.Println("Bot stopped")
}
