package svg

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/context-window-render/pkg/cwr/dsl"
	"github.com/go-go-golems/context-window-render/pkg/cwr/theme"
)

// Renderer produces SVG output from a Diagram.
type Renderer struct {
	theme *theme.Theme
}

// NewRenderer creates an SVG renderer with the given theme.
func NewRenderer(t *theme.Theme) *Renderer {
	return &Renderer{theme: t}
}

// Render converts a Diagram into a complete SVG document.
func (r *Renderer) Render(d *dsl.Diagram) (string, error) {
	windows := d.GetWindows()
	layout := d.Layout
	if layout == "" && len(windows) == 1 {
		layout = "single"
	} else if layout == "" {
		layout = "side-by-side"
	}

	windowWidth := 480
	if layout == "side-by-side" {
		windowWidth = 400
	}

	// Compute total canvas size
	totalWidth := 0
	if layout == "side-by-side" {
		for _, w := range windows {
			_, w_, _ := r.renderWindow(w, windowWidth)
			totalWidth += w_ + r.theme.WindowGap
		}
		totalWidth -= r.theme.WindowGap
	} else {
		for _, w := range windows {
			_, w_, _ := r.renderWindow(w, windowWidth)
			if w_ > totalWidth {
				totalWidth = w_
			}
		}
	}

	maxHeight := 0
	if layout == "side-by-side" {
		for _, w := range windows {
			_, _, h := r.renderWindow(w, windowWidth)
			if h > maxHeight {
				maxHeight = h
			}
		}
	} else {
		totalH := 0
		for _, w := range windows {
			_, _, h := r.renderWindow(w, windowWidth)
			totalH += h + r.theme.WindowGap
		}
		if len(windows) > 0 {
			maxHeight = totalH - r.theme.WindowGap
		}
	}

	canvasW := totalWidth + 2*r.theme.CanvasPadding
	canvasH := maxHeight + 2*r.theme.CanvasPadding

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" shape-rendering="crispEdges">`,
		canvasW, canvasH, canvasW, canvasH,
	))
	sb.WriteString("\n")
	sb.WriteString(`<rect width="100%" height="100%" fill="#FFFFFF"/>` + "\n")

	titleY := r.theme.CanvasPadding
	if d.Title != "" {
		sb.WriteString(fmt.Sprintf(
			`<text x="%d" y="%d" font-family="%s" font-size="16" font-weight="bold" fill="#000000">%s</text>`,
			r.theme.CanvasPadding, titleY+16, r.theme.FontFamily, escXML(d.Title),
		))
		titleY += 28
	}

	xOffset := r.theme.CanvasPadding
	yOffset := titleY

	for _, w := range windows {
		svg, w_, h := r.renderWindow(w, windowWidth)
		sb.WriteString(fmt.Sprintf(`<g transform="translate(%d,%d)">`, xOffset, yOffset))
		sb.WriteString(svg)
		sb.WriteString("</g>\n")

		if layout == "side-by-side" {
			xOffset += w_ + r.theme.WindowGap
		} else {
			yOffset += h + r.theme.WindowGap
		}
	}

	sb.WriteString("</svg>")
	return sb.String(), nil
}

// renderWindow renders a single Window and returns (svgContent, width, height).
func (r *Renderer) renderWindow(w dsl.Window, width int) (string, int, int) {
	th := r.theme
	innerWidth := width

	displayTitle := w.Title
	if displayTitle == "" {
		displayTitle = w.Name
	}
	showTitleBar := displayTitle != ""

	const baseContentHeight = 800

	contentHeight := 0
	regionRects := make([]regionLayout, len(w.Regions))

	y := 0
	for i, reg := range w.Regions {
		proportion := float64(reg.Size) / float64(w.Size)
		height := int(proportion * float64(baseContentHeight))
		if height < 24 {
			height = 24
		}
		regionRects[i] = regionLayout{
			x:      0,
			y:      y,
			width:  innerWidth,
			height: height,
			region: reg,
		}
		y += height + th.RegionGap
		contentHeight = y
	}
	if len(w.Regions) > 0 {
		contentHeight -= th.RegionGap
	}

	totalHeight := contentHeight
	titleBarH := 0
	if showTitleBar {
		titleBarH = th.TitleBarHeight
		totalHeight += titleBarH
		for i := range regionRects {
			regionRects[i].y += titleBarH
		}
	}

	var sb strings.Builder

	// Window outer border (drawn first, behind everything)
	sb.WriteString(fmt.Sprintf(
		`<rect x="0" y="0" width="%d" height="%d" fill="none" stroke="%s" stroke-width="%d"/>`,
		innerWidth, totalHeight, th.WindowBorderColor, th.WindowBorderWidth,
	))
	sb.WriteString("\n")

	// Title bar
	if showTitleBar {
		sb.WriteString(fmt.Sprintf(
			`<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`,
			th.WindowBorderWidth, th.WindowBorderWidth,
			innerWidth-2*th.WindowBorderWidth, titleBarH-th.WindowBorderWidth,
			th.TitleBarFill,
		))
		sb.WriteString("\n")
		// Title text left-aligned
		sb.WriteString(fmt.Sprintf(
			`<text x="8" y="%d" font-family="%s" font-size="11" font-weight="bold" fill="#FFFFFF">%s</text>`,
			titleBarH-7, th.FontFamily, escXML(displayTitle),
		))
		sb.WriteString("\n")
		// Size right-aligned in title bar
		sb.WriteString(fmt.Sprintf(
			`<text x="%d" y="%d" font-family="%s" font-size="9" fill="#AAAAAA" text-anchor="end">Size: %s</text>`,
			innerWidth-8, titleBarH-7, th.FontFamily, w.Size.String(),
		))
		sb.WriteString("\n")
		// Separator line below title bar
		sb.WriteString(fmt.Sprintf(
			`<line x1="0" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d"/>`,
			titleBarH, innerWidth, titleBarH, th.WindowBorderColor, th.WindowBorderWidth,
		))
		sb.WriteString("\n")
	} else {
		// Size label top-right inside border
		sb.WriteString(fmt.Sprintf(
			`<text x="%d" y="%d" font-family="%s" font-size="%g" fill="#888888" text-anchor="end">Size: %s</text>`,
			innerWidth-8, 14, th.FontFamily, th.FontSizeTiny, w.Size.String(),
		))
		sb.WriteString("\n")
	}

	// Render regions with separators
	for i, rl := range regionRects {
		sb.WriteString(r.renderRegion(rl, w))
		// Separator line between regions
		if i < len(regionRects)-1 {
			sepY := rl.y + rl.height + th.RegionGap/2
			sb.WriteString(fmt.Sprintf(
				`<line x1="1" y1="%d" x2="%d" y2="%d" stroke="#000000" stroke-width="1"/>`,
				sepY, innerWidth-1, sepY,
			))
			sb.WriteString("\n")
		}
	}

	return sb.String(), innerWidth, totalHeight
}

type regionLayout struct {
	x, y, width, height int
	region              dsl.Region
}

// renderRegion renders a single region (with subregions if any).
func (r *Renderer) renderRegion(rl regionLayout, w dsl.Window) string {
	th := r.theme
	cs := th.GetColor(rl.region.Color)
	hasSubs := len(rl.region.Subregions) > 0

	var sb strings.Builder

	// Overflow styling
	strokeDash := ""
	if rl.region.Overflow {
		strokeDash = fmt.Sprintf(` stroke-dasharray="%s"`, th.OverflowDashArray)
	}
	borderColor := cs.Border
	if rl.region.Overflow {
		borderColor = th.OverflowColor
	}

	fillAttr := cs.Fill
	if cs.Fill == "none" {
		fillAttr = "none"
	}

	// Region rectangle
	sb.WriteString(fmt.Sprintf(
		`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="%s" stroke-width="%d"%s rx="%d"/>`,
		rl.x, rl.y, rl.width, rl.height, fillAttr, borderColor, th.RegionBorderWidth, strokeDash, th.RegionCornerRadius,
	))
	sb.WriteString("\n")

	// ALWAYS show the region label (name + size), even with subregions
	{
		labelY := rl.y + 14
		label := rl.region.Name
		sb.WriteString(fmt.Sprintf(
			`<text x="%d" y="%d" font-family="%s" font-size="%g" fill="%s" font-weight="bold">%s</text>`,
			rl.x+8, labelY, th.FontFamily, th.FontSize, cs.Text, escXML(label),
		))
		sb.WriteString("\n")

		// Size line (always visible)
		sizeText := rl.region.Size.String()
		if w.ShowTokenCounts != nil && *w.ShowTokenCounts {
			sizeText = fmt.Sprintf("%d tkn", int(rl.region.Size))
		}
		if w.ShowPercentages != nil && *w.ShowPercentages {
			pct := float64(rl.region.Size) / float64(w.Size) * 100
			sizeText += fmt.Sprintf(" (%.1f%%)", pct)
		}
		sb.WriteString(fmt.Sprintf(
			`<text x="%d" y="%d" font-family="%s" font-size="%g" fill="%s">%s</text>`,
			rl.x+8, labelY+12, th.FontFamily, th.FontSizeSmall, cs.Subtext, sizeText,
		))
		sb.WriteString("\n")

		// Note (right-aligned)
		if rl.region.Note != "" {
			sb.WriteString(fmt.Sprintf(
				`<text x="%d" y="%d" font-family="%s" font-size="%g" fill="%s" text-anchor="end">%s</text>`,
				rl.x+rl.width-8, rl.y+14, th.FontFamily, th.FontSizeTiny, th.NoteColor, escXML(rl.region.Note),
			))
			sb.WriteString("\n")
		}
	}

	// Proportion fill bar (horizontal, below the text)
	if rl.height >= 38 {
		barY := rl.y + 30
		barHeight := 4
		if hasSubs {
			barY = rl.y + 30
			barHeight = 3
		}
		proportion := float64(rl.region.Size) / float64(w.Size)
		barWidth := int(proportion * float64(rl.width-16))
		if barWidth < 2 {
			barWidth = 2
		}
		barColor := cs.Text
		if cs.Fill == "none" {
			barColor = "#CCCCCC"
		}
		sb.WriteString(fmt.Sprintf(
			`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" opacity="0.3"/>`,
			rl.x+8, barY, barWidth, barHeight, barColor,
		))
		sb.WriteString("\n")
	}

	// Subregions
	if hasSubs {
		subTotal := dsl.Size(0)
		for _, sr := range rl.region.Subregions {
			subTotal += sr.Size
		}
		// Subregions start after the parent label area
		subStartY := rl.y + 38
		subAreaHeight := rl.height - 38
		if subAreaHeight < 20 {
			subAreaHeight = rl.height - 28
			subStartY = rl.y + 28
		}
		remainingHeight := subAreaHeight
		subY := subStartY

		for i, sr := range rl.region.Subregions {
			proportion := float64(sr.Size) / float64(subTotal)
			subHeight := int(proportion * float64(subAreaHeight))
			if i == len(rl.region.Subregions)-1 {
				subHeight = remainingHeight
			}
			remainingHeight -= subHeight

			scs := th.GetColor(sr.Color)
			subFill := scs.Fill
			if scs.Fill == "none" {
				subFill = "none"
			}

			// Subregion rectangle with left indent (tree-like)
			indent := 12
			sb.WriteString(fmt.Sprintf(
				`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="%s" stroke-width="0.5" rx="0"/>`,
				rl.x+indent, subY, rl.width-2*indent, subHeight, subFill, scs.Border,
			))
			sb.WriteString("\n")

			// Tree connector line (vertical)
			if subHeight >= 8 {
				sb.WriteString(fmt.Sprintf(
					`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="0.5"/>`,
					rl.x+6, subY+2, rl.x+6, subY+subHeight-2, scs.Border,
				))
				sb.WriteString("\n")
			}

			// Subregion label
			if subHeight >= 14 {
				labelY := subY + 11
				sb.WriteString(fmt.Sprintf(
					`<text x="%d" y="%d" font-family="%s" font-size="%g" fill="%s">%s</text>`,
					rl.x+indent+6, labelY, th.FontFamily, th.FontSizeTiny, scs.Text, escXML(sr.Name),
				))
				sb.WriteString("\n")
			}

			// Subregion size
			if subHeight >= 22 {
				sizeY := subY + 21
				sb.WriteString(fmt.Sprintf(
					`<text x="%d" y="%d" font-family="%s" font-size="%g" fill="%s">%s</text>`,
					rl.x+indent+6, sizeY, th.FontFamily, th.FontSizeTiny-1, scs.Subtext, sr.Size.String(),
				))
				sb.WriteString("\n")
			}

			subY += subHeight + th.SubregionGap
		}
	}

	// Warning (bottom-right of region)
	if rl.region.Warning != "" && rl.height >= 28 {
		sb.WriteString(fmt.Sprintf(
			`<text x="%d" y="%d" font-family="%s" font-size="%g" fill="%s" text-anchor="end" font-weight="bold">%s %s</text>`,
			rl.x+rl.width-8, rl.y+rl.height-6, th.FontFamily, th.FontSizeTiny, th.WarningColor, th.WarningSymbol, escXML(rl.region.Warning),
		))
		sb.WriteString("\n")
	}

	return sb.String()
}

// escXML escapes special XML characters.
func escXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
