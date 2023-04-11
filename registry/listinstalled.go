package registry

import (
	"github.com/alecthomas/kong"
	"github.com/olekukonko/tablewriter"
)

type ListInstalledRegistries struct {
	r *PluginRegistries
}

func (l *ListInstalledRegistries) BeforeApply() error {
	var err error
	l.r, err = LoadFromDisk()
	return err
}

func (l *ListInstalledRegistries) Run(ctx *kong.Context) error {
	registries := l.r.GetAll()

	table := tablewriter.NewWriter(ctx.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Registry Name", "URL"})
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor})
	table.SetColumnColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor}, tablewriter.Colors{tablewriter.FgYellowColor})
	table.SetHeaderLine(true)
	for n, r := range registries {
		table.Append([]string{n, r.URL.String()})
	}
	table.Render()
	return nil
}
