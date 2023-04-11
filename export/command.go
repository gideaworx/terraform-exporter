package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter-plugin/go-plugin"
	"github.com/gideaworx/terraform-exporter/runner"
)

type Command struct {
	OutputDirectory    string   `short:"o" type:"existingdir" default:"." help:"The directory to write the exported files to"`
	SkipProviderOutput bool     `default:"false" help:"If true, do not write the provider terraform file for the plugin"`
	CommandName        string   `arg:"" help:"The name of the command to use for export"`
	CommandArgs        []string `arg:"" help:"The args to pass to the command" passthrough:"true"`
}

const scriptHeader = `#!/usr/bin/env bash
	
set -e`

func (c *Command) Run(ctx *kong.Context) error {
	matching, err := runner.FindPluginsForCommand(c.CommandName)
	if err != nil {
		return err
	}

	if len(matching) == 0 {
		return fmt.Errorf("no installed plugins provide command %q", c.CommandName)
	}

	if len(matching) > 1 {
		options := []string{}
		for _, m := range matching {
			options = append(options, fmt.Sprintf("%s/%s", m[0], m[1]))
		}

		return fmt.Errorf("multiple plugins provide command %q. Valid choices are %q", c.CommandName, strings.Join(options, ", "))
	}

	pDef, err := runner.LoadPlugin(matching[0][0], nil)
	if err != nil {
		return err
	}
	defer pDef.Kill()

	impl, err := pDef.Plugin()
	if err != nil {
		return err
	}

	response, err := impl.Export(plugin.ExportPluginRequest{
		Name: matching[0][1],
		Request: plugin.ExportCommandRequest{
			OutputDirectory:    c.OutputDirectory,
			SkipProviderOutput: c.SkipProviderOutput,
			PluginArgs:         c.CommandArgs,
		},
	})
	if err != nil {
		return err
	}

	output, err := os.OpenFile(filepath.Join(c.OutputDirectory, "import.sh"), os.O_CREATE, 0o755)
	if err != nil {
		return err
	}
	defer output.Close()

	fmt.Fprintln(output, scriptHeader)
	for _, d := range response.Directives {
		fmt.Fprintf(output, "terraform import %s.%s %s\n", d.Resource, d.Name, d.ID)
	}

	return nil
}
