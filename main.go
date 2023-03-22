package main

import (
	"io"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/export"
	"github.com/gideaworx/terraform-exporter/help"
	"github.com/gideaworx/terraform-exporter/install"
	"github.com/gideaworx/terraform-exporter/list"
)

var cli struct {
	Export        *export.Command            `cmd:"" help:"Export data to terraform files"`
	InstallPlugin *install.Command           `cmd:"" help:"Install a plugin"`
	Help          *help.Command              `cmd:"" help:"Show help for a plugin's exporter command"`
	ListPlugins   *list.ListPluginsCommand   `cmd:"" help:"List installed plugins"`
	ListCommands  *list.ListExportersCommand `cmd:"" help:"List commands provided by installed plugins"`
}

func main() {
	ctx := kong.Parse(&cli)
	ctx.BindTo(ctx.Stdout, (*io.Writer)(nil))

	ctx.FatalIfErrorf(ctx.Run())
}
