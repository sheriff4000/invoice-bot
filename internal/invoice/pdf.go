package invoice

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/invoice-bot/internal/config"
)

// PDFGenerator creates invoice PDFs using configurable styling.
type PDFGenerator struct {
	style *config.Style
}

// NewPDFGenerator creates a new PDF generator with the given style.
func NewPDFGenerator(style *config.Style) *PDFGenerator {
	return &PDFGenerator{style: style}
}

// pdfCtx bundles the fpdf instance with a Unicode translator.
// fpdf's built-in fonts (helvetica, courier, times) use cp1252 encoding.
// The translator converts UTF-8 strings (e.g. containing £, €, ¥) to the
// correct single-byte encoding so they render properly in the PDF.
type pdfCtx struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

func newPDFCtx() *pdfCtx {
	pdf := fpdf.New("P", "mm", "A4", "")
	return &pdfCtx{
		pdf: pdf,
		tr:  pdf.UnicodeTranslatorFromDescriptor("cp1252"),
	}
}

// Generate creates a PDF from the given invoice and returns the raw bytes.
func (g *PDFGenerator) Generate(inv *Invoice) ([]byte, error) {
	s := g.style
	pc := newPDFCtx()
	pdf := pc.pdf
	pdf.SetMargins(s.PageMargin, s.PageMargin, s.PageMargin)
	pdf.SetAutoPageBreak(true, s.PageMargin+10)
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	contentW := pageW - 2*s.PageMargin

	y := s.PageMargin

	// ── Header: Logo + Company Name + "INVOICE" title ──
	y = g.renderHeader(pc, inv, contentW, y)

	// ── Divider line ──
	if s.ShowDividerLine {
		g.setColor(pdf, s.AccentColor, "draw")
		pdf.SetLineWidth(s.DividerLineWidth)
		pdf.Line(s.PageMargin, y, s.PageMargin+contentW, y)
		y += 4
	}

	// ── Invoice meta (number, date, due date, terms) + Recipient ──
	y = g.renderMetaAndRecipient(pc, inv, contentW, y)

	// ── Line items table ──
	if len(inv.LineItems) > 0 {
		y = g.renderLineItems(pc, inv, contentW, y)
	}

	// ── Totals ──
	y = g.renderTotals(pc, inv, contentW, y)

	// ── Payment details ──
	if inv.BankName != "" || inv.BankAccountNumber != "" || inv.BankSortCode != "" {
		y = g.renderPaymentDetails(pc, inv, contentW, y)
	}

	// ── Notes ──
	if inv.Notes != "" {
		y = g.renderNotes(pc, inv, contentW, y)
	}

	// ── Footer ──
	if s.FooterText != "" {
		g.renderFooter(pc, contentW)
	}

	_ = y // suppress unused warning

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *PDFGenerator) renderHeader(pc *pdfCtx, inv *Invoice, contentW, y float64) float64 {
	s := g.style
	pdf := pc.pdf
	tr := pc.tr

	// Company name
	if inv.Sender != nil && inv.Sender.Name != "" {
		g.setColor(pdf, s.PrimaryColor, "text")
		pdf.SetFont(s.FontFamily, "B", s.HeaderFontSize)
		pdf.SetXY(s.PageMargin, y)

		align := "L"
		if s.HeaderAlignment == "center" {
			align = "C"
		} else if s.HeaderAlignment == "right" {
			align = "R"
		}
		pdf.CellFormat(contentW, s.HeaderFontSize*0.5, tr(inv.Sender.Name), "", 0, align, false, 0, "")
		y += s.HeaderFontSize * 0.5
	}

	// Sender details (address, email, phone) — compact, muted
	if inv.Sender != nil {
		g.setColor(pdf, s.MutedColor, "text")
		pdf.SetFont(s.FontFamily, "", s.SmallFontSize)

		details := g.buildPartyDetails(inv.Sender)
		for _, line := range details {
			pdf.SetXY(s.PageMargin, y)
			pdf.CellFormat(contentW, s.SmallFontSize*0.5, tr(line), "", 0, "L", false, 0, "")
			y += s.SmallFontSize * 0.45
		}
	}

	y += 3

	// "INVOICE" title on the right side
	g.setColor(pdf, s.AccentColor, "text")
	pdf.SetFont(s.FontFamily, "B", s.HeaderFontSize*0.8)
	pdf.SetXY(s.PageMargin, s.PageMargin)
	pdf.CellFormat(contentW, s.HeaderFontSize*0.5, "INVOICE", "", 0, "R", false, 0, "")

	y += 3
	return y
}

func (g *PDFGenerator) renderMetaAndRecipient(pc *pdfCtx, inv *Invoice, contentW, y float64) float64 {
	s := g.style
	pdf := pc.pdf
	tr := pc.tr
	startY := y
	halfW := contentW / 2

	// Left column: Invoice metadata
	pdf.SetXY(s.PageMargin, y)
	metaY := y

	g.setColor(pdf, s.PrimaryColor, "text")
	pdf.SetFont(s.FontFamily, "B", s.SubheaderFSize)

	type metaRow struct {
		label, value string
	}
	rows := []metaRow{}

	if inv.InvoiceNumber != "" {
		rows = append(rows, metaRow{"Invoice #:", inv.InvoiceNumber})
	}
	rows = append(rows, metaRow{"Date:", inv.IssueDateOrDefault()})
	if inv.PaymentTerms != "" {
		rows = append(rows, metaRow{"Terms:", inv.PaymentTerms})
	}

	lineH := s.BodyFontSize * 0.55
	for _, row := range rows {
		g.setColor(pdf, s.MutedColor, "text")
		pdf.SetFont(s.FontFamily, "", s.BodyFontSize)
		pdf.SetXY(s.PageMargin, metaY)
		pdf.CellFormat(30, lineH, tr(row.label), "", 0, "L", false, 0, "")

		g.setColor(pdf, s.TextColor, "text")
		pdf.SetFont(s.FontFamily, "B", s.BodyFontSize)
		pdf.SetXY(s.PageMargin+30, metaY)
		pdf.CellFormat(halfW-30, lineH, tr(row.value), "", 0, "L", false, 0, "")
		metaY += lineH
	}

	// Right column: Recipient ("Bill To")
	recipY := startY
	if inv.Recipient != nil && inv.Recipient.Name != "" {
		rightX := s.PageMargin + halfW + 10

		g.setColor(pdf, s.MutedColor, "text")
		pdf.SetFont(s.FontFamily, "B", s.SmallFontSize)
		pdf.SetXY(rightX, recipY)
		pdf.CellFormat(halfW-10, lineH, "BILL TO", "", 0, "L", false, 0, "")
		recipY += lineH

		g.setColor(pdf, s.TextColor, "text")
		pdf.SetFont(s.FontFamily, "B", s.BodyFontSize)
		pdf.SetXY(rightX, recipY)
		pdf.CellFormat(halfW-10, lineH, tr(inv.Recipient.Name), "", 0, "L", false, 0, "")
		recipY += lineH

		details := g.buildPartyDetails(inv.Recipient)
		pdf.SetFont(s.FontFamily, "", s.BodyFontSize)
		for _, line := range details {
			pdf.SetXY(rightX, recipY)
			pdf.CellFormat(halfW-10, lineH, tr(line), "", 0, "L", false, 0, "")
			recipY += lineH
		}
	}

	y = metaY
	if recipY > y {
		y = recipY
	}
	y += 8
	return y
}

func (g *PDFGenerator) renderLineItems(pc *pdfCtx, inv *Invoice, contentW, y float64) float64 {
	s := g.style
	pdf := pc.pdf
	tr := pc.tr
	showDates := inv.HasLineItemDates()

	// Column widths — adaptive based on whether dates are present
	var descW, dateW, qtyW, unitPW, unitW, totalW float64
	if showDates {
		descW = contentW * 0.28
		dateW = contentW * 0.17
		qtyW = contentW * 0.10
		unitPW = contentW * 0.15
		unitW = contentW * 0.12
		totalW = contentW * 0.18
	} else {
		descW = contentW * 0.40
		qtyW = contentW * 0.15
		unitPW = contentW * 0.15
		unitW = contentW * 0.15
		totalW = contentW * 0.15
	}
	rowH := s.BodyFontSize * 0.6

	// Table header
	g.setColor(pdf, s.PrimaryColor, "fill")
	pdf.Rect(s.PageMargin, y, contentW, rowH+2, "F")

	pdf.SetFont(s.FontFamily, "B", s.BodyFontSize)
	pdf.SetTextColor(255, 255, 255) // White text on colored header

	x := s.PageMargin
	headerY := y + 1
	pdf.SetXY(x, headerY)
	pdf.CellFormat(descW, rowH, "  Description", "", 0, "L", false, 0, "")
	x += descW
	if showDates {
		pdf.SetXY(x, headerY)
		pdf.CellFormat(dateW, rowH, "Date", "", 0, "C", false, 0, "")
		x += dateW
	}
	pdf.SetXY(x, headerY)
	pdf.CellFormat(qtyW, rowH, "Qty", "", 0, "C", false, 0, "")
	x += qtyW
	pdf.SetXY(x, headerY)
	pdf.CellFormat(unitPW, rowH, "Unit Price", "", 0, "C", false, 0, "")
	x += unitPW
	pdf.SetXY(x, headerY)
	pdf.CellFormat(unitW, rowH, "Unit", "", 0, "C", false, 0, "")
	x += unitW
	pdf.SetXY(x, headerY)
	pdf.CellFormat(totalW, rowH, "Total", "", 0, "R", false, 0, "")

	y += rowH + 2

	// Table rows
	cur := inv.CurrencySymbol()
	for i, item := range inv.LineItems {
		// Stripe background
		if s.LineItemStripe && i%2 == 0 {
			g.setColor(pdf, s.StripeColor, "fill")
			pdf.Rect(s.PageMargin, y, contentW, rowH+2, "F")
		}

		g.setColor(pdf, s.TextColor, "text")
		pdf.SetFont(s.FontFamily, "", s.BodyFontSize)

		x = s.PageMargin
		rowY := y + 1

		pdf.SetXY(x, rowY)
		pdf.CellFormat(descW, rowH, tr("  "+item.Description), "", 0, "L", false, 0, "")
		x += descW

		if showDates {
			pdf.SetXY(x, rowY)
			pdf.CellFormat(dateW, rowH, tr(item.DateDisplay()), "", 0, "C", false, 0, "")
			x += dateW
		}

		pdf.SetXY(x, rowY)
		pdf.CellFormat(qtyW, rowH, formatNumber(item.Quantity), "", 0, "C", false, 0, "")
		x += qtyW

		pdf.SetXY(x, rowY)
		pdf.CellFormat(unitPW, rowH, tr(cur+formatMoney(item.UnitPrice)), "", 0, "C", false, 0, "")
		x += unitPW

		pdf.SetXY(x, rowY)
		unitLabel := item.Unit
		if unitLabel == "" {
			unitLabel = "-"
		}
		pdf.CellFormat(unitW, rowH, tr(unitLabel), "", 0, "C", false, 0, "")
		x += unitW

		pdf.SetXY(x, rowY)
		lineTotal := item.Quantity * item.UnitPrice
		pdf.CellFormat(totalW, rowH, tr(cur+formatMoney(lineTotal)), "", 0, "R", false, 0, "")

		y += rowH + 2
	}

	y += 3
	return y
}

func (g *PDFGenerator) renderTotals(pc *pdfCtx, inv *Invoice, contentW, y float64) float64 {
	s := g.style
	pdf := pc.pdf
	tr := pc.tr
	cur := inv.CurrencySymbol()
	lineH := s.BodyFontSize * 0.6

	// Totals are right-aligned
	labelX := s.PageMargin + contentW*0.6
	valueX := s.PageMargin + contentW*0.8
	valueW := contentW * 0.2

	// Subtotal
	g.setColor(pdf, s.MutedColor, "text")
	pdf.SetFont(s.FontFamily, "", s.BodyFontSize)
	pdf.SetXY(labelX, y)
	pdf.CellFormat(valueW, lineH, "Subtotal:", "", 0, "R", false, 0, "")
	g.setColor(pdf, s.TextColor, "text")
	pdf.SetXY(valueX, y)
	pdf.CellFormat(valueW, lineH, tr(cur+formatMoney(inv.Subtotal())), "", 0, "R", false, 0, "")
	y += lineH + 1

	// Discount
	if inv.DiscountAmount() > 0 {
		g.setColor(pdf, s.MutedColor, "text")
		pdf.SetFont(s.FontFamily, "", s.BodyFontSize)

		label := "Discount:"
		if inv.DiscountRate != nil {
			label = fmt.Sprintf("Discount (%.1f%%):", *inv.DiscountRate)
		}
		pdf.SetXY(labelX, y)
		pdf.CellFormat(valueW, lineH, label, "", 0, "R", false, 0, "")
		g.setColor(pdf, s.TextColor, "text")
		pdf.SetXY(valueX, y)
		pdf.CellFormat(valueW, lineH, tr("-"+cur+formatMoney(inv.DiscountAmount())), "", 0, "R", false, 0, "")
		y += lineH + 1
	}

	// Tax
	if inv.TaxRate != nil && *inv.TaxRate > 0 {
		g.setColor(pdf, s.MutedColor, "text")
		pdf.SetFont(s.FontFamily, "", s.BodyFontSize)
		pdf.SetXY(labelX, y)
		pdf.CellFormat(valueW, lineH, fmt.Sprintf("Tax (%.1f%%):", *inv.TaxRate), "", 0, "R", false, 0, "")
		g.setColor(pdf, s.TextColor, "text")
		pdf.SetXY(valueX, y)
		pdf.CellFormat(valueW, lineH, tr(cur+formatMoney(inv.TaxAmount())), "", 0, "R", false, 0, "")
		y += lineH + 1
	}

	// Divider before total
	y += 1
	g.setColor(pdf, s.AccentColor, "draw")
	pdf.SetLineWidth(0.3)
	pdf.Line(labelX, y, s.PageMargin+contentW, y)
	y += 3

	// Grand total
	g.setColor(pdf, s.PrimaryColor, "text")
	pdf.SetFont(s.FontFamily, "B", s.SubheaderFSize)
	pdf.SetXY(labelX, y)
	pdf.CellFormat(valueW, lineH+2, "TOTAL:", "", 0, "R", false, 0, "")
	pdf.SetXY(valueX, y)
	pdf.CellFormat(valueW, lineH+2, tr(cur+formatMoney(inv.Total())), "", 0, "R", false, 0, "")
	y += lineH + 8

	return y
}

func (g *PDFGenerator) renderPaymentDetails(pc *pdfCtx, inv *Invoice, contentW, y float64) float64 {
	s := g.style
	pdf := pc.pdf
	tr := pc.tr
	lineH := s.BodyFontSize * 0.55

	g.setColor(pdf, s.PrimaryColor, "text")
	pdf.SetFont(s.FontFamily, "B", s.SubheaderFSize)
	pdf.SetXY(s.PageMargin, y)
	pdf.CellFormat(contentW, lineH+2, "Payment Details", "", 0, "L", false, 0, "")
	y += lineH + 4

	type detailRow struct {
		label, value string
	}
	rows := []detailRow{}
	if inv.BankName != "" {
		rows = append(rows, detailRow{"Bank:", inv.BankName})
	}
	if inv.BankAccountNumber != "" {
		rows = append(rows, detailRow{"Account:", inv.BankAccountNumber})
	}
	if inv.BankSortCode != "" {
		rows = append(rows, detailRow{"Sort Code:", inv.BankSortCode})
	}

	for _, row := range rows {
		g.setColor(pdf, s.MutedColor, "text")
		pdf.SetFont(s.FontFamily, "", s.BodyFontSize)
		pdf.SetXY(s.PageMargin, y)
		pdf.CellFormat(25, lineH, row.label, "", 0, "L", false, 0, "")

		g.setColor(pdf, s.TextColor, "text")
		pdf.SetFont(s.FontFamily, "", s.BodyFontSize)
		pdf.SetXY(s.PageMargin+25, y)
		pdf.CellFormat(contentW-25, lineH, tr(row.value), "", 0, "L", false, 0, "")
		y += lineH
	}

	y += 6
	return y
}

func (g *PDFGenerator) renderNotes(pc *pdfCtx, inv *Invoice, contentW, y float64) float64 {
	s := g.style
	pdf := pc.pdf
	tr := pc.tr
	lineH := s.BodyFontSize * 0.55

	g.setColor(pdf, s.PrimaryColor, "text")
	pdf.SetFont(s.FontFamily, "B", s.SubheaderFSize)
	pdf.SetXY(s.PageMargin, y)
	pdf.CellFormat(contentW, lineH+2, "Notes", "", 0, "L", false, 0, "")
	y += lineH + 3

	g.setColor(pdf, s.TextColor, "text")
	pdf.SetFont(s.FontFamily, "", s.BodyFontSize)
	pdf.SetXY(s.PageMargin, y)
	pdf.MultiCell(contentW, lineH, tr(inv.Notes), "", "L", false)
	y = pdf.GetY() + 6

	return y
}

func (g *PDFGenerator) renderFooter(pc *pdfCtx, contentW float64) {
	s := g.style
	pdf := pc.pdf
	tr := pc.tr
	_, pageH := pdf.GetPageSize()

	g.setColor(pdf, s.MutedColor, "text")
	pdf.SetFont(s.FontFamily, "I", s.SmallFontSize)
	pdf.SetXY(s.PageMargin, pageH-s.PageMargin-5)
	pdf.CellFormat(contentW, s.SmallFontSize*0.5, tr(s.FooterText), "", 0, "C", false, 0, "")
}

// ── Helpers ──

func (g *PDFGenerator) setColor(pdf *fpdf.Fpdf, hex, target string) {
	r, gr, b, err := config.ParseHexColor(hex)
	if err != nil {
		r, gr, b = 0, 0, 0
	}
	switch target {
	case "text":
		pdf.SetTextColor(r, gr, b)
	case "fill":
		pdf.SetFillColor(r, gr, b)
	case "draw":
		pdf.SetDrawColor(r, gr, b)
	}
}

func (g *PDFGenerator) buildPartyDetails(p *Party) []string {
	var lines []string
	if p.Address != "" {
		// Split multi-line addresses
		for _, line := range strings.Split(p.Address, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	if p.Email != "" {
		lines = append(lines, p.Email)
	}
	if p.Phone != "" {
		lines = append(lines, p.Phone)
	}
	return lines
}

func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

func formatNumber(n float64) string {
	if n == float64(int(n)) {
		return fmt.Sprintf("%d", int(n))
	}
	return fmt.Sprintf("%.2f", n)
}
