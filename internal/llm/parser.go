package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/invoice-bot/internal/invoice"
	openai "github.com/sashabaranov/go-openai"
)

const systemPrompt = `You are an invoice data extraction assistant. The user will describe an invoice in natural language. Your job is to extract structured invoice data from their message and return it as JSON.

Rules:
- Extract all information the user provides. Do not invent information they did not mention.
- For line items, always extract description, quantity, and unit_price. If quantity is not mentioned, default to 1.
- If the user mentions a unit (hours, pieces, units, etc.), include it.
- If the user mentions when work was performed (a specific date or date range) for a line item, extract it into "date" and optionally "date_end". Use ISO 8601 format (YYYY-MM-DD).
- For dates, use ISO 8601 format (YYYY-MM-DD). If a relative date is given (e.g. "due in 30 days"), calculate from today's date which is %s.
- For tax_rate and discount_rate, use the percentage number (e.g. 10 for 10%%, not 0.1).
- For currency, use ISO 4217 code (e.g. "USD", "EUR", "GBP"). Default to "GBP" if not specified.
- If an invoice number is mentioned, include it. Otherwise omit it.
- Only include fields that the user explicitly or implicitly provides. Omit fields with no data.

Return ONLY valid JSON matching this schema:
{
  "invoice_number": "string (optional)",
  "issue_date": "YYYY-MM-DD (optional)",
  "due_date": "YYYY-MM-DD (optional)",
  "payment_terms": "string (optional)",
  "currency": "string (optional, ISO 4217)",
  "recipient": {
    "name": "string (optional)",
    "address": "string (optional)",
    "email": "string (optional)",
    "phone": "string (optional)"
  },
  "line_items": [
    {
      "description": "string",
      "quantity": number,
      "unit": "string (optional)",
      "unit_price": number,
      "date": "YYYY-MM-DD (optional, date work was performed)",
      "date_end": "YYYY-MM-DD (optional, end of date range if work spanned multiple days)"
    }
  ],
  "tax_rate": number (optional, percentage),
  "discount_rate": number (optional, percentage),
  "discount_flat": number (optional, fixed amount),
  "notes": "string (optional)"
}

Return ONLY the JSON object. No markdown, no explanation, no code fences.`

// Parser handles natural language to invoice conversion via OpenAI.
type Parser struct {
	client *openai.Client
	model  string
}

// NewParser creates a new LLM parser with the given API key and model.
func NewParser(apiKey, model string) *Parser {
	client := openai.NewClient(apiKey)
	return &Parser{
		client: client,
		model:  model,
	}
}

// Parse sends the user's natural language message to OpenAI and returns
// a structured Invoice.
func (p *Parser) Parse(ctx context.Context, message string) (*invoice.Invoice, error) {
	today := time.Now().Format("2006-01-02")
	prompt := fmt.Sprintf(systemPrompt, today)

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: prompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: message,
			},
		},
		Temperature: 0.1, // Low temperature for consistent structured output
	})
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	content := resp.Choices[0].Message.Content

	// Strip markdown code fences if present (just in case)
	content = stripCodeFences(content)

	inv := &invoice.Invoice{}
	if err := json.Unmarshal([]byte(content), inv); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nraw response: %s", err, content)
	}

	return inv, nil
}

// stripCodeFences removes markdown code fences from the response, in case
// the model wraps the JSON in ```json ... ``` blocks despite being told not to.
func stripCodeFences(s string) string {
	// Remove leading ```json or ```
	for _, prefix := range []string{"```json\n", "```json", "```\n", "```"} {
		if len(s) > len(prefix) && s[:len(prefix)] == prefix {
			s = s[len(prefix):]
			break
		}
	}
	// Remove trailing ```
	if len(s) > 3 && s[len(s)-3:] == "```" {
		s = s[:len(s)-3]
	}
	// Trim whitespace
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
