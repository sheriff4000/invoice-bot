package bot

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/invoice-bot/internal/config"
	"github.com/invoice-bot/internal/invoice"
	"github.com/invoice-bot/internal/llm"
)

// Handler manages Telegram bot interactions.
type Handler struct {
	bot    *tgbotapi.BotAPI
	auth   *Auth
	parser *llm.Parser
	pdfGen *invoice.PDFGenerator
	cfg    *config.Config
}

// NewHandler creates a new bot handler with all dependencies.
func NewHandler(bot *tgbotapi.BotAPI, auth *Auth, parser *llm.Parser, pdfGen *invoice.PDFGenerator, cfg *config.Config) *Handler {
	return &Handler{
		bot:    bot,
		auth:   auth,
		parser: parser,
		pdfGen: pdfGen,
		cfg:    cfg,
	}
}

// Run starts the bot's long-polling loop and handles incoming messages.
func (h *Handler) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := h.bot.GetUpdatesChan(u)

	log.Printf("Bot started as @%s", h.bot.Self.UserName)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down bot...")
			h.bot.StopReceivingUpdates()
			return ctx.Err()
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			go h.handleMessage(update.Message)
		}
	}
}

func (h *Handler) handleMessage(msg *tgbotapi.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Handle /whoami (always allowed, even for unauthorized users)
	if msg.IsCommand() && msg.Command() == "whoami" {
		text := fmt.Sprintf("Your Telegram user ID is: `%d`", userID)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = "Markdown"
		reply.ReplyToMessageID = msg.MessageID
		h.send(reply)
		return
	}

	// Auth check
	if !h.auth.IsAllowed(userID) {
		reply := tgbotapi.NewMessage(chatID, "Sorry, this bot is not available for your account.")
		reply.ReplyToMessageID = msg.MessageID
		h.send(reply)
		return
	}

	// Handle /start
	if msg.IsCommand() && msg.Command() == "start" {
		h.handleStart(chatID)
		return
	}

	// Handle /help
	if msg.IsCommand() && msg.Command() == "help" {
		h.handleHelp(chatID)
		return
	}

	// Skip other commands
	if msg.IsCommand() {
		reply := tgbotapi.NewMessage(chatID, "Unknown command. Use /help to see available commands.")
		reply.ReplyToMessageID = msg.MessageID
		h.send(reply)
		return
	}

	// Process invoice request
	h.handleInvoiceRequest(chatID, msg.MessageID, msg.Text)
}

func (h *Handler) handleStart(chatID int64) {
	text := `*Invoice Bot* 🧾

Send me a description of an invoice in plain language, and I'll generate a professional PDF for you.

*Example:*
_"Invoice ACME Corp for 3 hours of consulting at $150/hr with 10% tax, due in 30 days"_

*Commands:*
/help - Show usage instructions
/whoami - Show your Telegram user ID`

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "Markdown"
	h.send(reply)
}

func (h *Handler) handleHelp(chatID int64) {
	text := `*How to use this bot:*

Just send a message describing the invoice you need. I'll extract the details and generate a PDF.

*You can include:*
• Client/recipient name and address
• Line items with quantities and prices
• Tax rate
• Discount (percentage or flat amount)
• Due date or payment terms
• Invoice number
• Currency (USD, EUR, GBP, etc.)
• Notes

*Examples:*
_"Invoice John Smith at 123 Main St for 5 widgets at $20 each, invoice #1001, 8% tax, due Jan 15 2026"_

_"Bill to Acme Corp: 10 hours of development at €100/hr and 2 hours of design at €80/hr. 20% VAT. Net 14 days."_

_"Create invoice for Sarah Lee, 1 annual subscription $299, no tax, notes: Thank you for renewing!"_`

	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = "Markdown"
	h.send(reply)
}

func (h *Handler) handleInvoiceRequest(chatID int64, msgID int, text string) {
	// Send "processing" indicator
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	h.bot.Request(typing)

	// Parse with LLM
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inv, err := h.parser.Parse(ctx, text)
	if err != nil {
		log.Printf("LLM parse error: %v", err)
		reply := tgbotapi.NewMessage(chatID, "Sorry, I couldn't parse that invoice description. Please try again with more detail.")
		reply.ReplyToMessageID = msgID
		h.send(reply)
		return
	}

	// Merge company defaults from config
	h.applyDefaults(inv)

	// Generate PDF
	pdfBytes, err := h.pdfGen.Generate(inv)
	if err != nil {
		log.Printf("PDF generation error: %v", err)
		reply := tgbotapi.NewMessage(chatID, "Sorry, there was an error generating the PDF. Please try again.")
		reply.ReplyToMessageID = msgID
		h.send(reply)
		return
	}

	// Build filename
	filename := "invoice"
	if inv.InvoiceNumber != "" {
		filename = fmt.Sprintf("invoice_%s", inv.InvoiceNumber)
	}
	filename += ".pdf"

	// Send PDF
	fileBytes := tgbotapi.FileBytes{
		Name:  filename,
		Bytes: pdfBytes,
	}
	doc := tgbotapi.NewDocument(chatID, fileBytes)
	doc.ReplyToMessageID = msgID

	// Build caption with summary
	caption := h.buildCaption(inv)
	if len(caption) > 1024 {
		caption = caption[:1021] + "..."
	}
	doc.Caption = caption
	doc.ParseMode = "Markdown"

	h.send(doc)
}

// applyDefaults merges company info from environment config into the invoice.
func (h *Handler) applyDefaults(inv *invoice.Invoice) {
	// Set sender from config if not already set
	if inv.Sender == nil {
		inv.Sender = &invoice.Party{}
	}
	if inv.Sender.Name == "" && h.cfg.CompanyName != "" {
		inv.Sender.Name = h.cfg.CompanyName
	}
	if inv.Sender.Address == "" && h.cfg.CompanyAddress != "" {
		inv.Sender.Address = h.cfg.CompanyAddress
	}
	if inv.Sender.Email == "" && h.cfg.CompanyEmail != "" {
		inv.Sender.Email = h.cfg.CompanyEmail
	}
	if inv.Sender.Phone == "" && h.cfg.CompanyPhone != "" {
		inv.Sender.Phone = h.cfg.CompanyPhone
	}

	// Bank details from config
	if inv.BankName == "" && h.cfg.BankName != "" {
		inv.BankName = h.cfg.BankName
	}
	if inv.BankAccountNumber == "" && h.cfg.BankAccountNumber != "" {
		inv.BankAccountNumber = h.cfg.BankAccountNumber
	}
	if inv.BankSortCode == "" && h.cfg.BankSortCode != "" {
		inv.BankSortCode = h.cfg.BankSortCode
	}
}

func (h *Handler) buildCaption(inv *invoice.Invoice) string {
	cur := inv.CurrencySymbol()
	var b bytes.Buffer

	if inv.Recipient != nil && inv.Recipient.Name != "" {
		fmt.Fprintf(&b, "*To:* %s\n", inv.Recipient.Name)
	}
	if inv.InvoiceNumber != "" {
		fmt.Fprintf(&b, "*Invoice #:* %s\n", inv.InvoiceNumber)
	}
	fmt.Fprintf(&b, "*Total:* %s%.2f", cur, inv.Total())

	return b.String()
}

func (h *Handler) send(c tgbotapi.Chattable) {
	if _, err := h.bot.Send(c); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}
