package cmds

import (
	"context"
	"os"

	"github.com/go-go-golems/context-window-render/pkg/cwr/dsl"
	"github.com/go-go-golems/context-window-render/pkg/cwr/renderer/ascii"
	"github.com/go-go-golems/context-window-render/pkg/cwr/renderer/svg"
	"github.com/go-go-golems/context-window-render/pkg/cwr/theme"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
)

// RenderCommand renders a context window diagram from YAML to SVG/PNG/ASCII.
type RenderCommand struct {
	*cmds.CommandDescription
}

type RenderSettings struct {
	File   string `glazed:"file"`
	Format string `glazed:"format"`
	Out string `glazed:"out"`
}

func NewRenderCommand() (*RenderCommand, error) {
	glazedSection, err := settings.NewGlazedSchema(
		settings.WithOutputSectionOptions(
			schema.WithDefaults(map[string]interface{}{
				"output": "table",
			}),
		),
	)
	if err != nil {
		return nil, err
	}

	cmdDesc := cmds.NewCommandDescription(
		"render",
		cmds.WithShort("Render a context window diagram from YAML"),
		cmds.WithLong(`
Render a context window diagram defined in YAML to SVG, PNG, or ASCII format.

The YAML file uses a simple DSL that describes context windows as stacks of
proportional regions. See the examples/ directory for YAML samples.

Examples:
  cwr render --file examples/01-simple-window.yaml --format svg
  cwr render --file examples/03-rag-pipeline.yaml --format ascii
  cwr render --file examples/04-multi-window-comparison.yaml --format svg --output diagram.svg
`),
		cmds.WithFlags(
			fields.New(
				"file",
				fields.TypeString,
				fields.WithRequired(true),
				fields.WithHelp("Path to the YAML diagram file"),
			),
			fields.New(
				"format",
				fields.TypeString,
				fields.WithDefault("svg"),
				fields.WithHelp("Output format: svg, png, or ascii"),
			),
			fields.New(
				"out",
				fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Output file path (defaults to stdout)"),
			),
		),
		cmds.WithSections(glazedSection),
	)

	return &RenderCommand{CommandDescription: cmdDesc}, nil
}

func (c *RenderCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	settings := &RenderSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	// Load diagram
	diagram, err := dsl.LoadDiagram(settings.File)
	if err != nil {
		return err
	}

	th := theme.DefaultTheme()

	switch settings.Format {
	case "svg":
		r := svg.NewRenderer(th)
		result, err := r.Render(diagram)
		if err != nil {
			return err
		}
		if settings.Out != "" {
			return os.WriteFile(settings.Out, []byte(result), 0644)
		}
		_, err = os.Stdout.Write([]byte(result))
		return err

	case "ascii":
		r := ascii.NewRenderer(th)
		result, err := r.Render(diagram)
		if err != nil {
			return err
		}
		if settings.Out != "" {
			return os.WriteFile(settings.Out, []byte(result), 0644)
		}
		_, err = os.Stdout.Write([]byte(result))
		return err

	case "png":
		// PNG is rendered via SVG → PNG conversion (requires rsvg-convert or similar)
		r := svg.NewRenderer(th)
		svgContent, err := r.Render(diagram)
		if err != nil {
			return err
		}
		// Write temp SVG, convert, write PNG
		tmpFile, err := os.CreateTemp("", "cwr-*.svg")
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.WriteString(svgContent)
		tmpFile.Close()
		if err != nil {
			return err
		}
		outputPath := settings.Out
		if outputPath == "" {
			outputPath = "/dev/stdout"
		}
		// Use rsvg-convert for SVG→PNG
		// This is a simple approach; a production tool might use a Go SVG library
		row := types.NewRow(
			types.MRP("status", "png requires rsvg-convert (not yet integrated)"),
			types.MRP("svg_tmp", tmpFile.Name()),
		)
		return gp.AddRow(ctx, row)

	default:
		return gp.AddRow(ctx, types.NewRow(
			types.MRP("error", "unsupported format: "+settings.Format),
		))
	}
}
