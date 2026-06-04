package main

import (
	"github.com/go-go-golems/context-window-render/cmd/cwr/cmds"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "cwr",
	Short:   "Context Window Render — render context window diagrams from YAML",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

func main() {
	err := logging.AddLoggingSectionToRootCommand(rootCmd, "cwr")
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	// Render command
	renderCmd, err := cmds.NewRenderCommand()
	cobra.CheckErr(err)
	renderCobraCmd, err := cli.BuildCobraCommand(renderCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "cwr"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(renderCobraCmd)

	// Validate command
	validateCmd, err := cmds.NewValidateCommand()
	cobra.CheckErr(err)
	validateCobraCmd, err := cli.BuildCobraCommand(validateCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "cwr"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(validateCobraCmd)

	// Examples command
	examplesCmd, err := cmds.NewExamplesCommand()
	cobra.CheckErr(err)
	examplesCobraCmd, err := cli.BuildCobraCommand(examplesCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "cwr"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(examplesCobraCmd)

	// Serve command
	serveCmd, err := cmds.NewServeCommand()
	cobra.CheckErr(err)
	serveCobraCmd, err := cli.BuildCobraCommand(serveCmd,
		cli.WithParserConfig(cli.CobraParserConfig{AppName: "cwr"}),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(serveCobraCmd)

	_ = rootCmd.Execute()
}
