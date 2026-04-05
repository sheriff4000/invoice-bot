package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var defaultStyleYAML = `
primary_color: "#2C3E50"
accent_color: "#3498DB"
text_color: "#333333"
muted_color: "#7F8C8D"
font_family: "helvetica"
header_font_size: 24
subheader_font_size: 12
body_font_size: 10
small_font_size: 8
logo_max_width: 40
logo_max_height: 20
page_margin: 15
line_item_stripe: true
stripe_color: "#F0F4F8"
header_alignment: "left"
show_divider_line: true
divider_line_width: 0.5
footer_text: "Thank you for your business!"
`

// Style holds all visual styling configuration for the invoice PDF.
type Style struct {
	// Colors (hex strings)
	PrimaryColor string `yaml:"primary_color"`
	AccentColor  string `yaml:"accent_color"`
	TextColor    string `yaml:"text_color"`
	MutedColor   string `yaml:"muted_color"`

	// Fonts
	FontFamily     string  `yaml:"font_family"`
	HeaderFontSize float64 `yaml:"header_font_size"`
	SubheaderFSize float64 `yaml:"subheader_font_size"`
	BodyFontSize   float64 `yaml:"body_font_size"`
	SmallFontSize  float64 `yaml:"small_font_size"`

	// Layout (mm)
	LogoMaxWidth  float64 `yaml:"logo_max_width"`
	LogoMaxHeight float64 `yaml:"logo_max_height"`
	PageMargin    float64 `yaml:"page_margin"`

	// Table
	LineItemStripe bool   `yaml:"line_item_stripe"`
	StripeColor    string `yaml:"stripe_color"`

	// Header
	HeaderAlignment  string  `yaml:"header_alignment"`
	ShowDividerLine  bool    `yaml:"show_divider_line"`
	DividerLineWidth float64 `yaml:"divider_line_width"`

	// Footer
	FooterText string `yaml:"footer_text"`
}

// LoadStyle loads the style configuration. It first loads defaults, then
// overlays any user-provided style file on top.
func LoadStyle(path string) (*Style, error) {
	style := &Style{}
	if err := yaml.Unmarshal([]byte(defaultStyleYAML), style); err != nil {
		return nil, fmt.Errorf("parsing default style: %w", err)
	}

	// Overlay user style if path provided
	if path != "" {
		userData, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading style file %s: %w", path, err)
		}
		if err := yaml.Unmarshal(userData, style); err != nil {
			return nil, fmt.Errorf("parsing style file %s: %w", path, err)
		}
	}

	return style, nil
}

// ParseHexColor converts a hex color string like "#2C3E50" to RGB values (0-255).
func ParseHexColor(hex string) (r, g, b int, err error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %q", hex)
	}
	rr, err := strconv.ParseInt(hex[0:2], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	gg, err := strconv.ParseInt(hex[2:4], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	bb, err := strconv.ParseInt(hex[4:6], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	return int(rr), int(gg), int(bb), nil
}
