package list

import (
	"io"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/runner"
	"github.com/olekukonko/tablewriter"
)

type ListPluginsCommand struct {
}

func (l *ListPluginsCommand) Run(context *kong.Context, output io.Writer) error {
	plugins, err := runner.LoadInstalledBOMs()
	if err != nil {
		return err
	}

	tableData := [][]string{}
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	for _, p := range plugins {
		cmdNames := []string{}
		for _, c := range p.Provides {
			cmdNames = append(cmdNames, c.Name)
		}
		sort.Strings(cmdNames)
		tableData = append(tableData, []string{p.Name, p.Version.String(), strings.Join(cmdNames, ", ")})
	}

	if output == nil {
		output = context.Stdout
	}

	table := tablewriter.NewWriter(output)
	table.SetHeader([]string{"Plugin", "Version", "Provided Exporters"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
	)
	table.SetHeaderLine(true)
	table.SetBorder(true)
	table.AppendBulk(tableData)
	table.Render()

	return nil
}
