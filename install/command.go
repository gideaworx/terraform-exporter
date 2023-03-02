package install

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/runner"
	"github.com/hashicorp/go-hclog"
)

const (
	zip_header   = "\x50\x4b\x03\x04"
	gzip_header  = "\x1f\x8b"
	bzip2_header = "BZh"
	tar_header   = "\x75\x73\x74\x61\x72"
)

type decompressor func(io.Reader) (io.ReadCloser, error)
type decompressors map[string]decompressor

var ErrPluginAlreadyInstalled = errors.New("plugin already installed")

type Command struct {
	PluginNative  string        `short:"f" required:"true" xor:"source" help:"An executable file. Can be from the file system or a URL. If compressed via GZip or BZip2, it will be decompressed"`
	PluginJAR     string        `short:"j" required:"true" xor:"source" help:"An archive that be run via 'java -jar'. Can be from the file system or a URL. If compressed via GZip or BZip2, it will be decompressed first"`
	PluginNodeJS  string        `short:"n" required:"true" xor:"source" help:"The name of a NodeJS module to download via NPM. Node and npm must be installed. A version may be specified by specifying 'module@version'"`
	PluginPython  string        `short:"p" required:"true" xor:"source" help:"The name of a Python module to download via pip. Python v3+ and pip must be installed. A version may be specified by specifying 'module==version'"`
	logger        hclog.Logger  `kong:"-"`
	pluginHomeDir string        `kong:"-"`
	pluginDir     string        `kong:"-"`
	decompressors decompressors `kong:"-"`
	out           io.Writer     `kong:"-"`
	err           io.Writer     `kong:"-"`
	in            io.Reader     `kong:"-"`
}

func (i *Command) BeforeApply(ctx *kong.Context, logger hclog.Logger) {
	i.logger = logger

	i.out = ctx.Stdout
	i.err = ctx.Stderr

	i.in = strings.NewReader("")

	i.decompressors = map[string]decompressor{
		gzip_header:  i.gzipDecompressor,
		bzip2_header: i.bzip2Decompressor,
	}
}

func (i *Command) Run() error {
	pluginHome, err := runner.PluginHome()
	if err != nil {
		return err
	}
	i.pluginHomeDir = pluginHome

	fmt.Fprintf(i.out, "Installing %s\n\n", i.pluginDisplay())
	fmt.Fprintf(i.out, "Preparing installation ...\n")
	if err := i.prepare(); err != nil {
		return fmt.Errorf("could not prepare plugin: %w", err)
	}

	fmt.Fprintf(i.out, "\nInstalling ...\n")
	if err := i.install(); err != nil {
		return fmt.Errorf("could not install plugin: %w", err)
	}

	fmt.Fprintf(i.out, "\nFinalizing ...\n")
	if err := i.finalize(); err != nil {
		return fmt.Errorf("could not finalize plugin: %w", err)
	}

	return nil
}

func (i *Command) pluginDisplay() string {
	if i.PluginNative != "" {
		return fmt.Sprintf("native plugin from %q", i.PluginNative)
	}

	if i.PluginJAR != "" {
		return fmt.Sprintf("java archive plugin from %q", i.PluginJAR)
	}

	if i.PluginNodeJS != "" {
		return fmt.Sprintf("plugin %q from NPM registry", i.PluginNodeJS)
	}

	if i.PluginPython != "" {
		return fmt.Sprintf("plugin %q from PyPI registry", i.PluginPython)
	}

	return "!INVALID"
}
