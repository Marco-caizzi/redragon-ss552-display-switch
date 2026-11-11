// Package render draws the knob screen as SVG.
package render

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"

	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/display"
)

// Knob screen size, matching what other SS-552 knob plugins render.
const (
	Width  = 320
	Height = 130
)

// Status is what the knob is currently communicating.
type Status int

const (
	// Active: the shown mode is the one Windows is using.
	Active Status = iota
	// Pending: a different mode is selected and waits for a press to be applied.
	Pending
	// Applied: shown briefly after a successful switch.
	Applied
	// Failed: shown briefly after Windows rejected the switch.
	Failed
)

// View is everything needed to draw one knob screen.
type View struct {
	Mode   display.Mode
	Status Status
	// Remaining is the fraction (0..1) of the confirmation window left. Only used when Pending.
	Remaining float64
	// Seconds left to confirm, shown in the status line when Pending.
	Seconds int
}

const (
	colorBackground = "#000000"
	colorText       = "#ffffff"
	colorSubtle     = "#9ca3af"
	colorOn         = "#3b82f6"
	colorOff        = "#1f2937"
	colorOffStroke  = "#4b5563"
	colorActive     = "#22c55e"
	colorPending    = "#f59e0b"
	colorError      = "#ef4444"
)

// SVG renders the knob screen.
func SVG(v View) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, Width, Height, Width, Height)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, Width, Height, colorBackground)

	writeMonitors(&b, v.Mode)

	label := v.Mode.Label()
	if v.Status == Pending {
		// Chevrons hint that the knob can keep rotating.
		writeText(&b, 22, 84, 22, 700, colorPending, "‹")
		writeText(&b, Width-22, 84, 22, 700, colorPending, "›")
	}
	writeText(&b, Width/2, 84, 26, 800, colorText, label)

	statusText, statusColor := statusLine(v)
	writeText(&b, Width/2, 108, 15, 600, statusColor, statusText)

	if v.Status == Pending {
		remaining := math.Max(0, math.Min(1, v.Remaining))
		barWidth := int(math.Round(float64(Width-40) * remaining))
		fmt.Fprintf(&b, `<rect x="20" y="118" width="%d" height="5" rx="2.5" fill="%s"/>`, Width-40, colorOff)
		if barWidth > 0 {
			fmt.Fprintf(&b, `<rect x="20" y="118" width="%d" height="5" rx="2.5" fill="%s"/>`, barWidth, colorPending)
		}
	}

	b.WriteString(`</svg>`)
	return b.String()
}

func statusLine(v View) (string, string) {
	switch v.Status {
	case Pending:
		return fmt.Sprintf("Pulsá para aplicar · %ds", v.Seconds), colorPending
	case Applied:
		return "✓ Aplicado", colorActive
	case Failed:
		return "No se pudo aplicar", colorError
	default:
		if v.Mode == display.Unknown {
			return "Girá para elegir un modo", colorSubtle
		}
		return "● Activo", colorActive
	}
}

// writeMonitors draws two monitor glyphs showing which screens the mode uses.
func writeMonitors(b *strings.Builder, m display.Mode) {
	left, right := "1", "2"
	leftOn, rightOn := true, true
	switch m {
	case display.Internal:
		rightOn = false
	case display.External:
		leftOn = false
	case display.Clone:
		right = "1"
	case display.Unknown:
		leftOn, rightOn = false, false
		left, right = "?", "?"
	}
	const monitorWidth, monitorHeight, gap, top = 58, 36, 14, 12
	x0 := (Width - 2*monitorWidth - gap) / 2
	writeMonitor(b, x0, top, monitorWidth, monitorHeight, left, leftOn)
	writeMonitor(b, x0+monitorWidth+gap, top, monitorWidth, monitorHeight, right, rightOn)
	if m == display.Clone {
		// "=" between the screens: same content on both.
		cx := x0 + monitorWidth + gap/2
		fmt.Fprintf(b, `<rect x="%d" y="%d" width="8" height="2.5" fill="%s"/>`, cx-4, top+14, colorOn)
		fmt.Fprintf(b, `<rect x="%d" y="%d" width="8" height="2.5" fill="%s"/>`, cx-4, top+20, colorOn)
	}
}

func writeMonitor(b *strings.Builder, x, y, w, h int, label string, on bool) {
	fill, stroke, text := colorOn, colorOn, colorText
	if !on {
		fill, stroke, text = colorOff, colorOffStroke, colorOffStroke
	}
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="%s" stroke="%s" stroke-width="2"/>`, x, y, w, h, fill, stroke)
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="10" height="5" fill="%s"/>`, x+w/2-5, y+h, stroke)
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="24" height="3" rx="1.5" fill="%s"/>`, x+w/2-12, y+h+5, stroke)
	writeText(b, x+w/2, y+h/2+7, 19, 800, text, label)
	if !on {
		// Diagonal stroke marks a screen that will be turned off.
		fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2.5" stroke-linecap="round"/>`, x+6, y+h-6, x+w-6, y+6, colorOffStroke)
	}
}

func writeText(b *strings.Builder, x, y, size, weight int, color, value string) {
	fmt.Fprintf(b, `<text x="%d" y="%d" text-anchor="middle" font-family="Segoe UI, Arial, sans-serif" font-size="%d" font-weight="%d" fill="%s">%s</text>`,
		x, y, size, weight, color, escape(value))
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

// DataURI encodes an SVG for setImage.
func DataURI(svg string) string {
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}
