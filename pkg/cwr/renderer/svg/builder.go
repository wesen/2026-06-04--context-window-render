package svg

import (
	"fmt"
	"strings"
)

// Element is the base interface for all SVG elements.
type Element interface {
	Render() string
}

// BaseAttrs holds common SVG attributes shared by most elements.
type BaseAttrs struct {
	fill        string
	stroke      string
	strokeWidth *float64
	strokeDash  string
	opacity     *float64
	rx          *float64
	shapeRender string
	extra       string // catch-all for uncommon attributes
}

func (a BaseAttrs) renderAttrs() string {
	var parts []string
	if a.fill != "" {
		parts = append(parts, fmt.Sprintf(`fill="%s"`, a.fill))
	}
	if a.stroke != "" {
		parts = append(parts, fmt.Sprintf(`stroke="%s"`, a.stroke))
	}
	if a.strokeWidth != nil {
		parts = append(parts, fmt.Sprintf(`stroke-width="%g"`, *a.strokeWidth))
	}
	if a.strokeDash != "" {
		parts = append(parts, fmt.Sprintf(`stroke-dasharray="%s"`, a.strokeDash))
	}
	if a.opacity != nil {
		parts = append(parts, fmt.Sprintf(`opacity="%g"`, *a.opacity))
	}
	if a.rx != nil {
		parts = append(parts, fmt.Sprintf(`rx="%g"`, *a.rx))
	}
	if a.shapeRender != "" {
		parts = append(parts, fmt.Sprintf(`shape-rendering="%s"`, a.shapeRender))
	}
	if a.extra != "" {
		parts = append(parts, a.extra)
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// --- SVG root ---

// SVG is the root element of an SVG document.
type SVG struct {
	width    int
	height   int
	attrs    BaseAttrs
	children []Element
}

// NewSVG creates a new SVG root element.
func NewSVG(width, height int) *SVG {
	return &SVG{width: width, height: height}
}

func (s *SVG) Add(children ...Element) *SVG {
	s.children = append(s.children, children...)
	return s
}

func (s *SVG) Render() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d"%s>`,
		s.width, s.height, s.width, s.height,
		s.attrs.renderAttrs(),
	))
	sb.WriteString("\n")
	for _, c := range s.children {
		sb.WriteString(c.Render())
		sb.WriteString("\n")
	}
	sb.WriteString("</svg>")
	return sb.String()
}

// ShapeRendering sets the shape-rendering attribute on the SVG root.
func (s *SVG) ShapeRendering(v string) *SVG {
	s.attrs.shapeRender = v
	return s
}

// --- Rect ---

type Rect struct {
	x, y, width, height int
	attrs               BaseAttrs
}

// R creates a rectangle element.
func R(x, y, width, height int) *Rect {
	return &Rect{x: x, y: y, width: width, height: height}
}

func (r *Rect) Fill(v string) *Rect           { r.attrs.fill = v; return r }
func (r *Rect) Stroke(v string) *Rect         { r.attrs.stroke = v; return r }
func (r *Rect) StrokeWidth(v float64) *Rect   { r.attrs.strokeWidth = &v; return r }
func (r *Rect) StrokeDash(v string) *Rect     { r.attrs.strokeDash = v; return r }
func (r *Rect) Opacity(v float64) *Rect       { r.attrs.opacity = &v; return r }
func (r *Rect) Rx(v float64) *Rect            { r.attrs.rx = &v; return r }
func (r *Rect) ShapeRendering(v string) *Rect { r.attrs.shapeRender = v; return r }
func (r *Rect) Extra(v string) *Rect          { r.attrs.extra = v; return r }

func (r *Rect) Render() string {
	return fmt.Sprintf(
		`<rect x="%d" y="%d" width="%d" height="%d"%s/>`,
		r.x, r.y, r.width, r.height, r.attrs.renderAttrs(),
	)
}

// --- Line ---

type Line struct {
	x1, y1, x2, y2 int
	attrs          BaseAttrs
}

// L creates a line element.
func L(x1, y1, x2, y2 int) *Line {
	return &Line{x1: x1, y1: y1, x2: x2, y2: y2}
}

func (l *Line) Stroke(v string) *Line       { l.attrs.stroke = v; return l }
func (l *Line) StrokeWidth(v float64) *Line { l.attrs.strokeWidth = &v; return l }
func (l *Line) StrokeDash(v string) *Line   { l.attrs.strokeDash = v; return l }
func (l *Line) Opacity(v float64) *Line     { l.attrs.opacity = &v; return l }

func (l *Line) Render() string {
	return fmt.Sprintf(
		`<line x1="%d" y1="%d" x2="%d" y2="%d"%s/>`,
		l.x1, l.y1, l.x2, l.y2, l.attrs.renderAttrs(),
	)
}

// --- Text ---

type Text struct {
	x, y    int
	content string
	attrs   TextAttrs
}

type TextAttrs struct {
	fill       string
	fontFamily string
	fontSize   *float64
	fontWeight string
	textAnchor string
	extra      string
}

// T creates a text element.
func T(x, y int, content string) *Text {
	return &Text{x: x, y: y, content: content}
}

func (t *Text) Fill(v string) *Text       { t.attrs.fill = v; return t }
func (t *Text) FontFamily(v string) *Text { t.attrs.fontFamily = v; return t }
func (t *Text) FontSize(v float64) *Text  { t.attrs.fontSize = &v; return t }
func (t *Text) FontWeight(v string) *Text { t.attrs.fontWeight = v; return t }
func (t *Text) TextAnchor(v string) *Text { t.attrs.textAnchor = v; return t }
func (t *Text) Extra(v string) *Text      { t.attrs.extra = v; return t }

func (t *Text) Render() string {
	var parts []string
	if t.attrs.fill != "" {
		parts = append(parts, fmt.Sprintf(`fill="%s"`, t.attrs.fill))
	}
	if t.attrs.fontFamily != "" {
		parts = append(parts, fmt.Sprintf(`font-family="%s"`, t.attrs.fontFamily))
	}
	if t.attrs.fontSize != nil {
		parts = append(parts, fmt.Sprintf(`font-size="%g"`, *t.attrs.fontSize))
	}
	if t.attrs.fontWeight != "" {
		parts = append(parts, fmt.Sprintf(`font-weight="%s"`, t.attrs.fontWeight))
	}
	if t.attrs.textAnchor != "" {
		parts = append(parts, fmt.Sprintf(`text-anchor="%s"`, t.attrs.textAnchor))
	}
	if t.attrs.extra != "" {
		parts = append(parts, t.attrs.extra)
	}
	attrStr := ""
	if len(parts) > 0 {
		attrStr = " " + strings.Join(parts, " ")
	}
	return fmt.Sprintf(`<text x="%d" y="%d"%s>%s</text>`, t.x, t.y, attrStr, escXML(t.content))
}

// --- Group ---

type Group struct {
	transform string
	children  []Element
}

// G creates a group element.
func G() *Group {
	return &Group{}
}

func (g *Group) Translate(x, y int) *Group {
	g.transform = fmt.Sprintf("translate(%d,%d)", x, y)
	return g
}

func (g *Group) Add(children ...Element) *Group {
	g.children = append(g.children, children...)
	return g
}

func (g *Group) Render() string {
	var sb strings.Builder
	open := "<g"
	if g.transform != "" {
		open += fmt.Sprintf(` transform="%s"`, g.transform)
	}
	open += ">"
	sb.WriteString(open)
	sb.WriteString("\n")
	for _, c := range g.children {
		sb.WriteString(c.Render())
		sb.WriteString("\n")
	}
	sb.WriteString("</g>")
	return sb.String()
}

// --- Fragment (invisible container for grouping elements without <g>) ---

type Fragment struct {
	children []Element
}

// F creates a fragment (renders children inline, no wrapping element).
func F(children ...Element) *Fragment {
	return &Fragment{children: children}
}

func (f *Fragment) Render() string {
	var sb strings.Builder
	for i, c := range f.children {
		sb.WriteString(c.Render())
		if i < len(f.children)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// --- Helpers ---

// escXML escapes special XML characters.
func escXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
