package list

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/runner"
	"github.com/olekukonko/tablewriter"
)

type ListExportersCommand struct {
	Plugin   string `short:"p" optional:"true" help:"If specified, only show the exporters for the provided plugin"`
	Detailed bool   `short:"v" default:"false" help:"If set, show more detailed information about a plugin"`
}

func (l *ListExportersCommand) Run(context *kong.Context) error {
	plugins, err := runner.LoadInstalledBOMs()
	if err != nil {
		return err
	}

	pluginNames := []string{}
	formatters := []tablewriter.Colors{}

	headers := []string{"Command", "Version", "Description"}
	if l.Detailed {
		headers = append(headers, "Summary")
	}
	headers = append(headers, "Provided By")

	for range headers {
		formatters = append(formatters, tablewriter.Colors{tablewriter.Bold})
	}

	if l.Plugin != "" {
		var filtered runner.BillOfMaterials
		for _, p := range plugins {
			pluginNames = append(pluginNames, p.Name)
			if l.Plugin == p.Name {
				filtered = p
				break
			}
		}

		if filtered.Name == "" {
			return fmt.Errorf("could not find plugin %q. Installed plugins are %q", l.Plugin, strings.Join(pluginNames, ", "))
		}

		plugins = []runner.BillOfMaterials{filtered}
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	tableData := [][]string{}
	for _, p := range plugins {
		for _, c := range p.Provides {
			row := []string{c.Name, c.Version.String(), c.Description}
			if l.Detailed {
				row = append(row, c.Summary)
			}
			row = append(row, fmt.Sprintf("%s v%s", p.Name, p.Version))
			tableData = append(tableData, row)
		}
	}

	table := tablewriter.NewWriter(context.Stdout)
	table.SetHeader(headers)
	table.SetHeaderColor(formatters...)
	table.SetHeaderLine(true)
	table.SetBorder(true)
	table.AppendBulk(tableData)
	table.Render()

	return nil
}
