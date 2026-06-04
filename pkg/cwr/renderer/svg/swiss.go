package svg

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/context-window-render/pkg/cwr/dsl"
	"github.com/go-go-golems/context-window-render/pkg/cwr/theme"
)

// SwissRenderer renders context windows as a sparse Swiss-typography table:
// aligned text columns, generous whitespace, no boxes, no borders.
type SwissRenderer struct {
	theme   *theme.Theme
	palette SwissPalette
}

// SwissPalette maps semantic DSL color names into restrained text colors.
type SwissPalette struct {
	Name       string
	Background string
	Ink        string
	Muted      string
	Faint      string
	Accent     string
	Colors     map[string]string
}

// SwissPalettes returns the built-in Swiss table palettes.
func SwissPalettes() map[string]SwissPalette {
	return map[string]SwissPalette{
		"swiss": {
			Name:       "swiss",
			Background: "#FAFAF7",
			Ink:        "#111111",
			Muted:      "#6F6F68",
			Faint:      "#B8B8B0",
			Accent:     "#D64035", // restrained Swiss red
			Colors: map[string]string{
				"primary": "#111111", "secondary": "#4D4D48", "accent": "#D64035",
				"tool": "#2457A6", "tool-light": "#5278B8", "knowledge": "#2F7D56",
				"knowledge-light": "#609B79", "highlight": "#C58A12", "empty": "#9A9A92",
				"cycle": "#4D4D48", "think": "#2457A6", "act": "#C06014", "observe": "#2F7D56",
			},
		},
		"swiss-cool": {
			Name:       "swiss-cool",
			Background: "#F7F9FA",
			Ink:        "#101820",
			Muted:      "#63717A",
			Faint:      "#AEB8BF",
			Accent:     "#0067B1",
			Colors: map[string]string{
				"primary": "#101820", "secondary": "#3E515E", "accent": "#0067B1",
				"tool": "#0067B1", "tool-light": "#4B91C8", "knowledge": "#00866E",
				"knowledge-light": "#58A995", "highlight": "#B48300", "empty": "#8B979E",
				"cycle": "#3E515E", "think": "#0067B1", "act": "#B85B00", "observe": "#00866E",
			},
		},
		"swiss-warm": {
			Name:       "swiss-warm",
			Background: "#FBF7F0",
			Ink:        "#1B1714",
			Muted:      "#756C63",
			Faint:      "#BDB4AA",
			Accent:     "#B33A2B",
			Colors: map[string]string{
				"primary": "#1B1714", "secondary": "#5B5149", "accent": "#B33A2B",
				"tool": "#7A4E9E", "tool-light": "#9B7ABA", "knowledge": "#47734B",
				"knowledge-light": "#769879", "highlight": "#B97800", "empty": "#998E82",
				"cycle": "#5B5149", "think": "#7A4E9E", "act": "#B05A1D", "observe": "#47734B",
			},
		},
	}
}

// NewSwissRenderer creates a Swiss table renderer. Unknown palette names fall back to "swiss".
func NewSwissRenderer(t *theme.Theme, paletteName string) *SwissRenderer {
	palettes := SwissPalettes()
	p, ok := palettes[paletteName]
	if !ok {
		p = palettes["swiss"]
	}
	return &SwissRenderer{theme: t, palette: p}
}

// Render renders a diagram as a Swiss typography SVG table.
func (r *SwissRenderer) Render(d *dsl.Diagram) (string, error) {
	windows := d.GetWindows()
	rows := r.collectRows(windows)

	width := 1040
	top := 52
	rowH := 24
	windowGap := 46
	lineChartH := 8
	height := top + len(rows)*rowH + len(windows)*(windowGap+lineChartH) + 48
	if d.Title != "" {
		height += 30
	}

	root := NewSVG(width, height).Add(R(0, 0, width, height).Fill(r.palette.Background))
	y := top
	if d.Title != "" {
		root.Add(
			T(48, y, strings.ToUpper(d.Title)).Fill(r.palette.Ink).FontFamily(r.font()).FontSize(r.titleSize()).FontWeight("700").Extra(`letter-spacing="1.2"`),
		)
		y += 34
	}

	for wi, w := range windows {
		name := w.Title
		if name == "" {
			name = w.Name
		}
		if name == "" {
			name = fmt.Sprintf("WINDOW %d", wi+1)
		}

		root.Add(
			T(48, y, strings.ToUpper(name)).Fill(r.palette.Ink).FontFamily(r.font()).FontSize(r.titleSize()).FontWeight("700").Extra(`letter-spacing="0.4"`),
			T(920, y, fmt.Sprintf("%s TOKENS", w.Size.String())).Fill(r.palette.Muted).FontFamily(r.font()).FontSize(r.metaSize()).TextAnchor("end").Extra(`letter-spacing="1"`),
		)
		y += 28

		root.Add(
			T(48, y, "REGION").Fill(r.palette.Faint).FontFamily(r.font()).FontSize(r.metaSize()).Extra(`letter-spacing="1.8"`),
			T(650, y, "TOKENS").Fill(r.palette.Faint).FontFamily(r.font()).FontSize(r.metaSize()).TextAnchor("end").Extra(`letter-spacing="1.8"`),
			T(740, y, "PCT").Fill(r.palette.Faint).FontFamily(r.font()).FontSize(r.metaSize()).TextAnchor("end").Extra(`letter-spacing="1.8"`),
			T(880, y, "TYPE").Fill(r.palette.Faint).FontFamily(r.font()).FontSize(r.metaSize()).Extra(`letter-spacing="1.8"`),
		)
		y += 22

		for _, row := range r.windowRows(w) {
			color := r.color(row.color)
			nameX := 48 + row.indent*28
			pct := float64(row.size) / float64(w.Size) * 100
			root.Add(
				T(nameX, y, row.name).Fill(color).FontFamily(r.font()).FontSize(r.bodySize()).FontWeight("400"),
				T(650, y, row.size.String()).Fill(r.palette.Ink).FontFamily(r.font()).FontSize(r.bodySize()).TextAnchor("end"),
				T(740, y, fmt.Sprintf("%.1f", pct)).Fill(r.palette.Muted).FontFamily(r.font()).FontSize(r.bodySize()).TextAnchor("end"),
				T(880, y, r.typeLabel(row.color)).Fill(color).FontFamily(r.font()).FontSize(r.metaSize()).Extra(`letter-spacing="0.8"`),
			)
			if row.note != "" {
				root.Add(T(1010, y, row.note).Fill(r.palette.Muted).FontFamily(r.font()).FontSize(r.metaSize()).TextAnchor("end"))
			}
			if row.warning != "" {
				root.Add(T(1010, y, "! "+row.warning).Fill(r.palette.Accent).FontFamily(r.font()).FontSize(r.metaSize()).TextAnchor("end"))
			}
			y += rowH
		}

		root.Add(r.lineChart(w, 48, y+4, 872, lineChartH))
		y += windowGap + lineChartH
	}

	root.Add(
		T(48, height-24, "CONTEXT WINDOW RENDER · SWISS TABLE · "+strings.ToUpper(r.palette.Name)).Fill(r.palette.Faint).FontFamily(r.font()).FontSize(r.metaSize()).Extra(`letter-spacing="1.4"`),
	)

	return root.Render(), nil
}

type swissRow struct {
	name    string
	size    dsl.Size
	color   string
	indent  int
	note    string
	warning string
}

func (r *SwissRenderer) collectRows(windows []dsl.Window) []swissRow {
	rows := []swissRow{}
	for _, w := range windows {
		rows = append(rows, r.windowRows(w)...)
	}
	return rows
}

func (r *SwissRenderer) windowRows(w dsl.Window) []swissRow {
	rows := []swissRow{}
	for _, reg := range w.Regions {
		rows = append(rows, swissRow{name: reg.Name, size: reg.Size, color: reg.Color, indent: 0, note: reg.Note, warning: reg.Warning})
		for _, sr := range reg.Subregions {
			rows = append(rows, swissRow{name: sr.Name, size: sr.Size, color: sr.Color, indent: 1, note: sr.Note, warning: sr.Warning})
		}
	}
	return rows
}

func (r *SwissRenderer) lineChart(w dsl.Window, x, y, width, height int) Element {
	children := []Element{
		R(x, y, width, height).Fill(r.palette.Faint).Opacity(0.18),
	}

	cursor := x
	remainingWidth := width
	for i, reg := range w.Regions {
		segmentW := int(float64(reg.Size) / float64(w.Size) * float64(width))
		if segmentW < 1 && reg.Size > 0 {
			segmentW = 1
		}
		if i == len(w.Regions)-1 {
			segmentW = remainingWidth
		}
		if segmentW < 0 {
			segmentW = 0
		}
		children = append(children,
			R(cursor, y, segmentW, height).Fill(r.color(reg.Color)),
		)
		cursor += segmentW
		remainingWidth -= segmentW
	}

	return F(children...)
}

func (r *SwissRenderer) font() string {
	return "Helvetica Neue, Helvetica, Arial, sans-serif"
}

func (r *SwissRenderer) titleSize() float64 { return 22 }
func (r *SwissRenderer) bodySize() float64  { return 13 }
func (r *SwissRenderer) metaSize() float64  { return 10 }

func (r *SwissRenderer) color(name string) string {
	if c, ok := r.palette.Colors[name]; ok {
		return c
	}
	return r.palette.Ink
}

func (r *SwissRenderer) typeLabel(name string) string {
	if name == "" {
		return "DEFAULT"
	}
	return strings.ToUpper(strings.ReplaceAll(name, "-", " "))
}
