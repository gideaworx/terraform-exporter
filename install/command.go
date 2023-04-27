package install

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/registry"
	"github.com/gideaworx/terraform-exporter/runner"
)

var ErrPluginAlreadyInstalled = errors.New("plugin already installed")

type Command struct {
	LocalFile     bool                       `short:"f" help:"If true, treat the plugin-name arg as the path to a local file"`
	Registry      string                     `short:"r" default:"default" help:"The name of the registry to install the plugin from"`
	PluginVersion string                     `help:"The version to install from the registry. Ignored if --local-file is set"`
	PluginName    string                     `arg:"" help:"The name of the plugin to install if --registry is true, or the path to the executable plugin if --local-file is set"`
	pluginHomeDir string                     `kong:"-"`
	out           io.Writer                  `kong:"-"`
	err           io.Writer                  `kong:"-"`
	in            io.Reader                  `kong:"-"`
	r             *registry.PluginRegistries `kong:"-"`
}

func (i *Command) BeforeApply(ctx *kong.Context) error {
	i.out = ctx.Stdout
	i.err = ctx.Stderr

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

	fmt.Fprintf(i.out, "Installing %s\n\n", i.PluginName)

	if i.LocalFile {
		return i.localInstall()
	}

	return i.registryInstall()
}
