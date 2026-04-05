package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Required
	TelegramBotToken string
	OpenAIAPIKey     string

	// Optional - OpenAI
	OpenAIModel string

	// Optional - Auth
	AllowedUserIDs []int64

	// Optional - Company defaults
	CompanyName    string
	CompanyAddress string
	CompanyEmail   string
	CompanyPhone   string

	// Optional - Bank details
	BankName    string
	BankAccountNumber string
	BankSortCode string

	// Optional - Paths
	CompanyLogoPath  string
	InvoiceStylePath string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	allowedIDs, err := parseUserIDs(os.Getenv("ALLOWED_USER_IDS"))
	if err != nil {
		return nil, fmt.Errorf("invalid ALLOWED_USER_IDS: %w", err)
	}

	return &Config{
		TelegramBotToken: token,
		OpenAIAPIKey:     apiKey,
		OpenAIModel:      model,
		AllowedUserIDs:   allowedIDs,
		CompanyName:      os.Getenv("COMPANY_NAME"),
		CompanyAddress:   os.Getenv("COMPANY_ADDRESS"),
		CompanyEmail:     os.Getenv("COMPANY_EMAIL"),
		CompanyPhone:     os.Getenv("COMPANY_PHONE"),
		BankName:         os.Getenv("COMPANY_BANK_NAME"),
		BankAccountNumber:      os.Getenv("COMPANY_BANK_ACCOUNT_NUMBER"),
		BankSortCode:      os.Getenv("COMPANY_BANK_SORT_CODE"),
		CompanyLogoPath:  os.Getenv("COMPANY_LOGO_PATH"),
		InvoiceStylePath: os.Getenv("INVOICE_STYLE_PATH"),
	}, nil
}

func parseUserIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID %q: %w", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
