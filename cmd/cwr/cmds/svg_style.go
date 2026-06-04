package cmds

import (
	"fmt"

	"github.com/go-go-golems/context-window-render/pkg/cwr/dsl"
	"github.com/go-go-golems/context-window-render/pkg/cwr/renderer/svg"
	"github.com/go-go-golems/context-window-render/pkg/cwr/theme"
)

// renderSVG selects one of the supported SVG renderers.
func renderSVG(diagram *dsl.Diagram, th *theme.Theme, style string) (string, error) {
	switch style {
	case "", "boxed", "mac", "macintosh":
		return svg.NewRenderer(th).Render(diagram)
	case "swiss", "swiss-cool", "swiss-warm":
		return svg.NewSwissRenderer(th, style).Render(diagram)
	default:
		return "", fmt.Errorf("unsupported SVG style %q (expected boxed, swiss, swiss-cool, swiss-warm)", style)
	}
}
