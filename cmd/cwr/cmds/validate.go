package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ValidateCommand validates a context window YAML file.
type ValidateCommand struct {
	*cmds.CommandDescription
}

type ValidateSettings struct {
	File string `glazed:"file"`
}

func NewValidateCommand() (*ValidateCommand, error) {
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
		"validate",
		cmds.WithShort("Validate a context window YAML file"),
		cmds.WithLong(`
Validate the structure and consistency of a context window YAML diagram.

Checks:
- Required fields present
- Region sizes sum doesn't exceed window size
- Subregion sizes don't exceed parent region size
- Size values are non-negative
- No duplicate region names

Examples:
  cwr validate --file examples/01-simple-window.yaml
  cwr validate --file examples/03-rag-pipeline.yaml
`),
		cmds.WithFlags(
			fields.New(
				"file",
				fields.TypeString,
				fields.WithRequired(true),
				fields.WithHelp("Path to the YAML diagram file"),
			),
		),
		cmds.WithSections(glazedSection),
	)

	return &ValidateCommand{CommandDescription: cmdDesc}, nil
}

func (c *ValidateCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	settings := &ValidateSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	diagram, err := dsl.LoadDiagram(settings.File)
	if err != nil {
		row := types.NewRow(
			types.MRP("file", settings.File),
			types.MRP("valid", false),
			types.MRP("error", err.Error()),
		)
		return gp.AddRow(ctx, row)
	}

	// Additional checks beyond basic validation
	warnings := []string{}
	for _, w := range diagram.GetWindows() {
		totalUsed := dsl.Size(0)
		for _, r := range w.Regions {
			totalUsed += r.Size
		}
		usage := float64(totalUsed) / float64(w.Size) * 100
		if usage > 95 {
			warnings = append(warnings, fmt.Sprintf("Window usage at %.1f%% (near capacity)", usage))
		}
		if usage < 10 {
			warnings = append(warnings, fmt.Sprintf("Window usage at %.1f%% (mostly empty)", usage))
		}
	}

	row := types.NewRow(
		types.MRP("file", settings.File),
		types.MRP("valid", true),
		types.MRP("windows", len(diagram.GetWindows())),
		types.MRP("warnings", strings.Join(warnings, "; ")),
	)
	return gp.AddRow(ctx, row)
}

// ExamplesCommand lists and optionally renders the bundled examples.
type ExamplesCommand struct {
	*cmds.CommandDescription
}

type ExamplesSettings struct {
	Render bool   `glazed:"render"`
	Format string `glazed:"format"`
}

func NewExamplesCommand() (*ExamplesCommand, error) {
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
		"examples",
		cmds.WithShort("List and render bundled example diagrams"),
		cmds.WithLong(`
List all bundled example YAML diagrams, or render them to a given format.

Examples:
  cwr examples
  cwr examples --render --format ascii
  cwr examples --render --format svg
`),
		cmds.WithFlags(
			fields.New(
				"render",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Render examples instead of just listing them"),
			),
			fields.New(
				"format",
				fields.TypeString,
				fields.WithDefault("ascii"),
				fields.WithHelp("Output format when rendering: svg or ascii"),
			),
		),
		cmds.WithSections(glazedSection),
	)

	return &ExamplesCommand{CommandDescription: cmdDesc}, nil
}

func (c *ExamplesCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	settings := &ExamplesSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	examplesDir := "examples"
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		return fmt.Errorf("failed to read examples directory: %w", err)
	}

	th := theme.DefaultTheme()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(examplesDir, entry.Name())

		if !settings.Render {
			row := types.NewRow(
				types.MRP("name", strings.TrimSuffix(entry.Name(), ".yaml")),
				types.MRP("file", path),
			)
			if err := gp.AddRow(ctx, row); err != nil {
				return err
			}
			continue
		}

		// Render the example
		diagram, err := dsl.LoadDiagram(path)
		if err != nil {
			row := types.NewRow(
				types.MRP("name", entry.Name()),
				types.MRP("error", err.Error()),
			)
			if err := gp.AddRow(ctx, row); err != nil {
				return err
			}
			continue
		}

		var result string
		switch settings.Format {
		case "svg":
			r := svg.NewRenderer(th)
			result, err = r.Render(diagram)
		case "ascii":
			r := ascii.NewRenderer(th)
			result, err = r.Render(diagram)
		}
		if err != nil {
			row := types.NewRow(
				types.MRP("name", entry.Name()),
				types.MRP("error", err.Error()),
			)
			if err := gp.AddRow(ctx, row); err != nil {
				return err
			}
			continue
		}

		row := types.NewRow(
			types.MRP("name", strings.TrimSuffix(entry.Name(), ".yaml")),
			types.MRP("format", settings.Format),
			types.MRP("output", result),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	return nil
}
