package ascii

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/context-window-render/pkg/cwr/dsl"
	"github.com/go-go-golems/context-window-render/pkg/cwr/theme"
)

// Renderer produces ASCII art output from a Diagram.
type Renderer struct {
	theme *theme.Theme
}

// NewRenderer creates an ASCII renderer with the given theme.
func NewRenderer(t *theme.Theme) *Renderer {
	return &Renderer{theme: t}
}

// Render converts a Diagram into ASCII art.
func (r *Renderer) Render(d *dsl.Diagram) (string, error) {
	windows := d.GetWindows()
	layout := d.Layout
	if layout == "" && len(windows) == 1 {
		layout = "single"
	} else if layout == "" {
		layout = "stacked"
	}

	var sb strings.Builder

	if d.Title != "" {
		sb.WriteString(fmt.Sprintf("  %s\n", d.Title))
		sb.WriteString("  " + strings.Repeat("─", len(d.Title)) + "\n\n")
	}

	for i, w := range windows {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(r.renderWindow(w))
	}

	return sb.String(), nil
}

// renderWindow renders a single Window as ASCII.
func (r *Renderer) renderWindow(w dsl.Window) string {
	width := 52
	innerWidth := width - 4 // account for "│ " and " │"

	var sb strings.Builder

	// Title bar
	titleText := ""
	if w.Title != "" {
		titleText = fmt.Sprintf(" %s ", w.Title)
	}
	titlePad := innerWidth - len(titleText)
	if titlePad < 0 {
		titlePad = 0
	}
	leftPad := titlePad / 2
	rightPad := titlePad - leftPad

	sb.WriteString("  ┌" + strings.Repeat("─", innerWidth) + "┐\n")
	if titleText != "" {
		sb.WriteString(fmt.Sprintf("  │%s%s%s│\n",
			strings.Repeat(" ", leftPad),
			titleText,
			strings.Repeat(" ", rightPad),
		))
		sb.WriteString("  ├" + strings.Repeat("─", innerWidth) + "┤\n")
	}

	// Size label
	sb.WriteString(fmt.Sprintf("  │ Size: %s%*s│\n", w.Size.String(), innerWidth-7-len(w.Size.String()), ""))

	// Regions
	for _, reg := range w.Regions {
		sb.WriteString(r.renderRegion(reg, w, innerWidth))
	}

	// Footer with overflow indicator
	sb.WriteString("  └" + strings.Repeat("─", innerWidth) + "┘\n")

	return sb.String()
}

// renderRegion renders a single region as ASCII.
func (r *Renderer) renderRegion(reg dsl.Region, w dsl.Window, innerWidth int) string {
	var sb strings.Builder

	// Determine visual character
	fillChar := " "
	borderChar := "─"
	switch reg.Color {
	case "accent":
		fillChar = "█"
	case "empty":
		fillChar = "·"
	case "highlight":
		fillChar = "▓"
	case "overflow":
		fillChar = "▒"
	default:
		fillChar = "░"
	}

	// Region header line
	label := reg.Name
	if reg.Icon != "" {
		label = reg.Icon + " " + label
	}
	overflowMark := ""
	if reg.Overflow {
		overflowMark = " ⚠ OVERFLOW"
	}

	headerText := fmt.Sprintf(" %s%s ", label, overflowMark)
	if len(headerText) > innerWidth {
		headerText = headerText[:innerWidth]
	}
	headerPad := innerWidth - len(headerText)
	if headerPad < 0 {
		headerPad = 0
	}

	sb.WriteString("  ├" + strings.Repeat(borderChar, innerWidth) + "┤\n")
	sb.WriteString(fmt.Sprintf("  │%s%s│\n", headerText, strings.Repeat(" ", headerPad)))

	// Size line
	sizeText := fmt.Sprintf(" %s", reg.Size.String())
	if w.ShowPercentages != nil && *w.ShowPercentages {
		pct := float64(reg.Size) / float64(w.Size) * 100
		sizeText += fmt.Sprintf(" (%.1f%%)", pct)
	}
	if reg.Warning != "" {
		sizeText += fmt.Sprintf("  ⚠ %s", reg.Warning)
	}
	if reg.Note != "" {
		sizeText += fmt.Sprintf("  [%s]", reg.Note)
	}
	if len(sizeText) > innerWidth {
		sizeText = sizeText[:innerWidth]
	}
	sizePad := innerWidth - len(sizeText)
	if sizePad < 0 {
		sizePad = 0
	}
	sb.WriteString(fmt.Sprintf("  │%s%s│\n", sizeText, strings.Repeat(" ", sizePad)))

	// Visual fill line (shows proportional size with fill characters)
	proportion := float64(reg.Size) / float64(w.Size)
	fillCount := int(proportion * float64(innerWidth))
	if fillCount > innerWidth {
		fillCount = innerWidth
	}
	emptyCount := innerWidth - fillCount
	fillLine := strings.Repeat(fillChar, fillCount) + strings.Repeat(" ", emptyCount)
	sb.WriteString(fmt.Sprintf("  │%s│\n", fillLine))

	// Subregions
	if len(reg.Subregions) > 0 {
		for _, sr := range reg.Subregions {
			sb.WriteString(r.renderSubregion(sr, reg, innerWidth))
		}
	}

	return sb.String()
}

// renderSubregion renders a subregion as ASCII.
func (r *Renderer) renderSubregion(sr dsl.Region, parent dsl.Region, innerWidth int) string {
	var sb strings.Builder

	indent := "  "

	label := fmt.Sprintf("   ├ %s", sr.Name)
	if len(label) > innerWidth {
		label = label[:innerWidth]
	}
	labelPad := innerWidth - len(label)
	if labelPad < 0 {
		labelPad = 0
	}
	sb.WriteString(fmt.Sprintf("  │%s%s│\n", label, strings.Repeat(" ", labelPad)))

	sizeText := fmt.Sprintf("   │ %s", sr.Size.String())
	if len(sizeText) > innerWidth {
		sizeText = sizeText[:innerWidth]
	}
	sizePad := innerWidth - len(sizeText)
	if sizePad < 0 {
		sizePad = 0
	}
	sb.WriteString(fmt.Sprintf("  │%s%s│\n", sizeText, strings.Repeat(" ", sizePad)))

	// Mini fill bar
	subProp := float64(sr.Size) / float64(parent.Size)
	fillCount := int(subProp * float64(innerWidth-6))
	if fillCount > innerWidth-6 {
		fillCount = innerWidth - 6
	}
	fillChar := "░"
	switch sr.Color {
	case "tool-light":
		fillChar = "░"
	case "knowledge-light":
		fillChar = "▒"
	case "think":
		fillChar = "▓"
	case "act":
		fillChar = "▓"
	case "observe":
		fillChar = "░"
	default:
		fillChar = "░"
	}
	fillLine := indent + strings.Repeat(fillChar, fillCount) + strings.Repeat(" ", innerWidth-6-fillCount)
	if len(fillLine) > innerWidth {
		fillLine = fillLine[:innerWidth]
	}
	fillPad := innerWidth - len(fillLine)
	sb.WriteString(fmt.Sprintf("  │%s%s│\n", fillLine, strings.Repeat(" ", fillPad)))

	return sb.String()
}
