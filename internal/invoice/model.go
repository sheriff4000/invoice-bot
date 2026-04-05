package invoice

import (
	"fmt"
	"time"
)

// Invoice represents the full data model for an invoice.
// All fields are optional — the PDF generator will only render sections
// that have data.
type Invoice struct {
	// Invoice metadata
	InvoiceNumber string `json:"invoice_number,omitempty"`
	IssueDate     string `json:"issue_date,omitempty"`
	DueDate       string `json:"due_date,omitempty"`
	PaymentTerms  string `json:"payment_terms,omitempty"`
	Currency      string `json:"currency,omitempty"`

	// Sender / company info
	Sender *Party `json:"sender,omitempty"`

	// Recipient / client info
	Recipient *Party `json:"recipient,omitempty"`

	// Line items
	LineItems []LineItem `json:"line_items,omitempty"`

	// Tax
	TaxRate *float64 `json:"tax_rate,omitempty"` // percentage, e.g. 10 for 10%

	// Discount
	DiscountRate *float64 `json:"discount_rate,omitempty"` // percentage
	DiscountFlat *float64 `json:"discount_flat,omitempty"` // fixed amount

	// Bank / payment details
	BankName          string `json:"bank_name,omitempty"`
	BankAccountNumber string `json:"bank_account_number,omitempty"`
	BankSortCode      string `json:"bank_sort_code,omitempty"`

	// Notes
	Notes string `json:"notes,omitempty"`
}

// Party represents either the sender (company) or recipient (client).
type Party struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
}

// LineItem represents a single line item on the invoice.
type LineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit,omitempty"` // e.g. "hours", "units", "pieces"
	UnitPrice   float64 `json:"unit_price"`
	Date        string  `json:"date,omitempty"`     // ISO 8601 date, e.g. "2026-03-15"
	DateEnd     string  `json:"date_end,omitempty"` // ISO 8601 end date for ranges
}

// DateDisplay returns a human-friendly date string for the line item.
// Returns "" if no date is set.
// Single date: "15 Mar 2026"
// Date range, same month: "15 - 19 Mar 2026"
// Date range, different months: "15 Mar - 2 Apr 2026"
// Date range, different years: "15 Dec 2025 - 2 Jan 2026"
func (li *LineItem) DateDisplay() string {
	if li.Date == "" {
		return ""
	}

	start, err := time.Parse("2006-01-02", li.Date)
	if err != nil {
		return li.Date // fallback to raw string
	}

	if li.DateEnd == "" {
		return start.Format("2 Jan 2006")
	}

	end, err := time.Parse("2006-01-02", li.DateEnd)
	if err != nil {
		return start.Format("2 Jan 2006")
	}

	if start.Year() != end.Year() {
		return start.Format("2 Jan 2006") + " - " + end.Format("2 Jan 2006")
	}
	if start.Month() != end.Month() {
		return start.Format("2 Jan") + " - " + end.Format("2 Jan 2006")
	}
	return fmt.Sprintf("%d - %s", start.Day(), end.Format("2 Jan 2006"))
}

// HasDates returns true if any line item in the invoice has a date set.
func (inv *Invoice) HasLineItemDates() bool {
	for _, item := range inv.LineItems {
		if item.Date != "" {
			return true
		}
	}
	return false
}

// Subtotal returns the sum of all line item totals.
func (inv *Invoice) Subtotal() float64 {
	total := 0.0
	for _, item := range inv.LineItems {
		total += item.Quantity * item.UnitPrice
	}
	return total
}

// TaxAmount returns the computed tax amount based on the subtotal and tax rate.
func (inv *Invoice) TaxAmount() float64 {
	if inv.TaxRate == nil {
		return 0
	}
	return inv.Subtotal() * (*inv.TaxRate / 100)
}

// DiscountAmount returns the total discount amount.
func (inv *Invoice) DiscountAmount() float64 {
	amount := 0.0
	if inv.DiscountRate != nil {
		amount += inv.Subtotal() * (*inv.DiscountRate / 100)
	}
	if inv.DiscountFlat != nil {
		amount += *inv.DiscountFlat
	}
	return amount
}

// Total returns the grand total: subtotal - discount + tax.
func (inv *Invoice) Total() float64 {
	return inv.Subtotal() - inv.DiscountAmount() + inv.TaxAmount()
}

// CurrencySymbol returns the currency symbol for display, defaulting to "£" (GBP).
func (inv *Invoice) CurrencySymbol() string {
	switch inv.Currency {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "JPY":
		return "¥"
	case "CHF":
		return "CHF "
	case "CAD":
		return "CA$"
	case "AUD":
		return "A$"
	case "GBP", "":
		return "£"
	default:
		return inv.Currency + " "
	}
}

// IssueDateOrDefault returns the issue date, defaulting to today.
func (inv *Invoice) IssueDateOrDefault() string {
	if inv.IssueDate != "" {
		return inv.IssueDate
	}
	return time.Now().Format("2006-01-02")
}
