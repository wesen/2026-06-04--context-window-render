package svg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-go-golems/context-window-render/pkg/cwr/dsl"
	"github.com/go-go-golems/context-window-render/pkg/cwr/theme"
)

// Renderer produces SVG output from a Diagram using the fluent builder API.
type Renderer struct {
	theme *theme.Theme
}

// NewRenderer creates an SVG renderer with the given theme.
func NewRenderer(t *theme.Theme) *Renderer {
	return &Renderer{theme: t}
}

// winSize holds the rendered dimensions of a window.
type winSize struct {
	width  int
	height int
}

// Render converts a Diagram into a complete SVG document string.
func (r *Renderer) Render(d *dsl.Diagram) (string, error) {
	windows := d.GetWindows()
	layout := r.layoutMode(d)

	windowWidth := r.windowWidth(layout)

	// Compute all window SVG subtrees and their dimensions
	results := make([]winSize, len(windows))
	elements := make([]Element, len(windows))
	for i, w := range windows {
		elem, w_, h := r.buildWindow(w, windowWidth)
		elements[i] = elem
		results[i] = winSize{width: w_, height: h}
	}

	canvasW, canvasH := r.canvasSize(results, layout)

	root := NewSVG(canvasW, canvasH).
		ShapeRendering("crispEdges").
		Add(
			R(0, 0, canvasW, canvasH).Fill("#FFFFFF"),
		)

	// Title
	titleY := r.theme.CanvasPadding
	if d.Title != "" {
		root.Add(
			T(r.theme.CanvasPadding, titleY+16, d.Title).
				FontFamily(r.theme.FontFamily).
				FontSize(16).FontWeight("bold").Fill("#000000"),
		)
		titleY += 28
	}

	// Place windows
	xOff := r.theme.CanvasPadding
	yOff := titleY
	for i, wr := range results {
		g := G().Translate(xOff, yOff).Add(elements[i])
		root.Add(g)

		if layout == "side-by-side" {
			xOff += wr.width + r.theme.WindowGap
		} else {
			yOff += wr.height + r.theme.WindowGap
		}
	}

	return root.Render(), nil
}

func (r *Renderer) layoutMode(d *dsl.Diagram) string {
	if d.Layout != "" {
		return d.Layout
	}
	if len(d.GetWindows()) == 1 {
		return "single"
	}
	return "side-by-side"
}

func (r *Renderer) windowWidth(layout string) int {
	if layout == "side-by-side" {
		return 400
	}
	return 480
}

func (r *Renderer) canvasSize(results []winSize, layout string) (int, int) {
	totalWidth := 0
	maxHeight := 0

	if layout == "side-by-side" {
		for _, wr := range results {
			totalWidth += wr.width + r.theme.WindowGap
		}
		if len(results) > 0 {
			totalWidth -= r.theme.WindowGap
		}
		for _, wr := range results {
			if wr.height > maxHeight {
				maxHeight = wr.height
			}
		}
	} else {
		for _, wr := range results {
			if wr.width > totalWidth {
				totalWidth = wr.width
			}
			maxHeight += wr.height + r.theme.WindowGap
		}
		if len(results) > 0 {
			maxHeight -= r.theme.WindowGap
		}
	}

	return totalWidth + 2*r.theme.CanvasPadding,
		maxHeight + 2*r.theme.CanvasPadding
}

// buildWindow constructs the SVG elements for a single Window.
// Returns (element tree, pixel width, pixel height).
func (r *Renderer) buildWindow(w dsl.Window, width int) (Element, int, int) {
	th := r.theme
	displayTitle := w.Title
	if displayTitle == "" {
		displayTitle = w.Name
	}
	showTitleBar := displayTitle != ""

	const baseContentHeight = 800

	// Layout regions: compute (y, height) for each
	type regionLayout struct {
		y      int
		height int
		region dsl.Region
	}
	layout := make([]regionLayout, len(w.Regions))
	y := 0
	for i, reg := range w.Regions {
		proportion := float64(reg.Size) / float64(w.Size)
		h := int(proportion * float64(baseContentHeight))
		if h < 24 {
			h = 24
		}
		layout[i] = regionLayout{y: y, height: h, region: reg}
		y += h + th.RegionGap
	}
	contentHeight := y
	if len(w.Regions) > 0 {
		contentHeight -= th.RegionGap
	}

	titleBarH := 0
	if showTitleBar {
		titleBarH = th.TitleBarHeight
		for i := range layout {
			layout[i].y += titleBarH
		}
	}
	totalHeight := contentHeight + titleBarH

	// Build element tree
	children := []Element{}

	// Window outer border
	children = append(children,
		R(0, 0, width, totalHeight).
			Fill("#FFFFFF").
			Stroke(th.WindowBorderColor).
			StrokeWidth(float64(th.WindowBorderWidth)),
	)

	// Title bar
	if showTitleBar {
		children = append(children,
			// Title bar fill
			R(th.WindowBorderWidth, th.WindowBorderWidth,
				width-2*th.WindowBorderWidth, titleBarH-th.WindowBorderWidth).
				Fill(th.TitleBarFill),
			// Title text (left)
			T(8, titleBarH-7, displayTitle).
				Fill("#FFFFFF").
				FontFamily(th.FontFamily).FontSize(11).FontWeight("bold"),
			// Size badge (right)
			T(width-8, titleBarH-7, "Size: "+w.Size.String()).
				Fill("#AAAAAA").
				FontFamily(th.FontFamily).FontSize(9).TextAnchor("end"),
			// Separator line below title bar
			L(0, titleBarH, width, titleBarH).
				Stroke(th.WindowBorderColor).
				StrokeWidth(float64(th.WindowBorderWidth)),
		)
	} else {
		children = append(children,
			T(width-8, 14, "Size: "+w.Size.String()).
				Fill("#888888").
				FontFamily(th.FontFamily).FontSize(th.FontSizeTiny).TextAnchor("end"),
		)
	}

	// Regions + separators
	for i, rl := range layout {
		children = append(children, r.buildRegion(rl.region, rl.y, rl.height, width, w))
		if i < len(layout)-1 {
			sepY := rl.y + rl.height + th.RegionGap/2
			children = append(children,
				L(1, sepY, width-1, sepY).Stroke("#000000").StrokeWidth(1),
			)
		}
	}

	return F(children...), width, totalHeight
}

// buildRegion constructs SVG elements for a single region.
func (r *Renderer) buildRegion(reg dsl.Region, y, height, width int, w dsl.Window) Element {
	th := r.theme
	cs := th.GetColor(reg.Color)
	hasSubs := len(reg.Subregions) > 0

	children := []Element{}

	// --- Region rectangle ---
	rect := R(0, y, width, height).
		Fill(cs.Fill).
		Stroke(r.regionStroke(reg, cs)).
		StrokeWidth(float64(th.RegionBorderWidth)).
		Rx(float64(th.RegionCornerRadius))
	if reg.Overflow {
		rect = rect.StrokeDash(th.OverflowDashArray)
	}
	children = append(children, rect)

	// --- Region label (always visible) ---
	children = append(children,
		T(8, y+14, reg.Name).
			Fill(cs.Text).
			FontFamily(th.FontFamily).
			FontSize(th.FontSize).
			FontWeight("bold"),
	)

	// --- Size label ---
	sizeText := reg.Size.String()
	if w.ShowTokenCounts != nil && *w.ShowTokenCounts {
		sizeText = fmt.Sprintf("%d tkn", int(reg.Size))
	}
	if w.ShowPercentages != nil && *w.ShowPercentages {
		pct := float64(reg.Size) / float64(w.Size) * 100
		sizeText += fmt.Sprintf(" (%.1f%%)", pct)
	}
	children = append(children,
		T(8, y+26, sizeText).
			Fill(cs.Subtext).
			FontFamily(th.FontFamily).
			FontSize(th.FontSizeSmall),
	)

	// --- Note (right-aligned) ---
	if reg.Note != "" {
		children = append(children,
			T(width-8, y+14, reg.Note).
				Fill(th.NoteColor).
				FontFamily(th.FontFamily).
				FontSize(th.FontSizeTiny).
				TextAnchor("end"),
		)
	}

	// --- Proportion fill bar ---
	if height >= 38 {
		barY := y + 30
		barH := 4
		if hasSubs {
			barH = 3
		}
		proportion := float64(reg.Size) / float64(w.Size)
		barW := int(proportion * float64(width-16))
		if barW < 2 {
			barW = 2
		}
		barColor := cs.Text
		if cs.Fill == "none" {
			barColor = "#CCCCCC"
		}
		children = append(children,
			R(8, barY, barW, barH).
				Fill(barColor).
				Opacity(0.3),
		)
	}

	// --- Subregions ---
	if hasSubs {
		subTotal := dsl.Size(0)
		for _, sr := range reg.Subregions {
			subTotal += sr.Size
		}
		subStartY := y + 38
		subAreaHeight := height - 38
		if subAreaHeight < 20 {
			subStartY = y + 28
			subAreaHeight = height - 28
		}
		remaining := subAreaHeight
		subY := subStartY
		indent := 12

		for i, sr := range reg.Subregions {
			prop := float64(sr.Size) / float64(subTotal)
			subH := int(prop * float64(subAreaHeight))
			if i == len(reg.Subregions)-1 {
				subH = remaining
			}
			remaining -= subH

			scs := th.GetColor(sr.Color)
			subFill := scs.Fill
			if scs.Fill == "none" {
				subFill = "none"
			}

			// Subregion rectangle
			children = append(children,
				R(indent, subY, width-2*indent, subH).
					Fill(subFill).
					Stroke(scs.Border).
					StrokeWidth(0.5),
			)

			// Tree connector
			if subH >= 8 {
				children = append(children,
					L(6, subY+2, 6, subY+subH-2).
						Stroke(scs.Border).StrokeWidth(0.5),
				)
			}

			// Subregion label
			if subH >= 14 {
				children = append(children,
					T(indent+6, subY+11, sr.Name).
						Fill(scs.Text).
						FontFamily(th.FontFamily).
						FontSize(th.FontSizeTiny),
				)
			}

			// Subregion size
			if subH >= 22 {
				children = append(children,
					T(indent+6, subY+21, sr.Size.String()).
						Fill(scs.Subtext).
						FontFamily(th.FontFamily).
						FontSize(th.FontSizeTiny-1),
				)
			}

			subY += subH + th.SubregionGap
		}
	}

	// --- Warning ---
	if reg.Warning != "" && height >= 28 {
		children = append(children,
			T(width-8, y+height-6, th.WarningSymbol+" "+reg.Warning).
				Fill(th.WarningColor).
				FontFamily(th.FontFamily).
				FontSize(th.FontSizeTiny).
				TextAnchor("end").
				FontWeight("bold"),
		)
	}

	return F(children...)
}

func (r *Renderer) regionStroke(reg dsl.Region, cs theme.ColorSet) string {
	if reg.Overflow {
		return r.theme.OverflowColor
	}
	return cs.Border
}

// sortedNames is a helper for the serve command.
func sortedNames(m map[string]struct{}) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Suppress unused import
var _ = fmt.Sprintf
var _ = strings.ReplaceAll
