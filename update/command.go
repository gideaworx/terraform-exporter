package update

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/registry"
	"github.com/gideaworx/terraform-exporter/runner"
)

var ErrPluginNewer = errors.New("installed plugin is newer than the candidate")

type Command struct {
	LocalFile       bool                       `short:"f" help:"If set, treat the plugin-name arg as the path to a local file"`
	Registry        string                     `short:"r" default:"default" help:"The name of the registry to install the plugin from"`
	PluginVersion   string                     `help:"The version to install from the registry. Ignored if --local-file is set"`
	AllowDowngrades bool                       `default:"false" help:"If set, allow an upgrade even if the new version is lower than the installed version"`
	Install         bool                       `default:"false" help:"If set, install the plugin if the plugin isn't already installed"`
	PluginName      string                     `arg:"" help:"The name of the plugin to install if --registry is true, or the path to the executable plugin if --local-file is set"`
	pluginHomeDir   string                     `kong:"-"`
	ctx             *kong.Context              `kong:"-"`
	in              io.Reader                  `kong:"-"`
	r               *registry.PluginRegistries `kong:"-"`
}

func (i *Command) BeforeApply(ctx *kong.Context) error {
	i.ctx = ctx

	i.in = strings.NewReader("")

	var err error
	i.r, err = registry.LoadFromDisk()

	return err
}

func (i *Command) Run() error {
	pluginHome, err := runner.PluginHome()
	if err != nil {
		return err
	}
	i.pluginHomeDir = pluginHome

	version := "the latest version"
	if i.PluginVersion != "" {
		version = fmt.Sprintf("version %s", i.PluginVersion)
	}

	fmt.Fprintf(i.ctx.Stdout, "Updating %s to %s\n\n", i.PluginName, version)

	if i.LocalFile {
		return i.localUpdate()
	}

	return i.registryUpdate()
}
