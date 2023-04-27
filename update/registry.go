package update

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter-plugin-registry/registry"
	"github.com/gideaworx/terraform-exporter/install"
	"github.com/gideaworx/terraform-exporter/remove"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (c *Command) registryUpdate() error {
	i := &install.Command{
		LocalFile:     false,
		PluginName:    c.PluginName,
		Registry:      c.Registry,
		PluginVersion: c.PluginVersion,
	}
	if err := i.BeforeApply(c.ctx); err != nil {
		return err
	}

	r := &remove.Command{
		PluginName:     c.PluginName,
		NonInteractive: true,
	}
	r.BeforeApply(&kong.Context{
		Kong: &kong.Kong{
			Stdout: io.Discard,
			Stderr: io.Discard,
		},
	}, strings.NewReader(""))

	bom, err := runner.LoadPluginBOM(filepath.Base(c.PluginName))
	if errors.Is(err, runner.ErrPluginNotFound) {
		if c.Install {
			return i.Run()
		}

		return err
	}

	if c.AllowDowngrades {
		// we don't have to version check here
		err = r.Run()
		if err != nil {
			return err
		}

		return i.Run()
	}

	reg := c.r.Get(c.Registry)
	if reg == nil {
		return fmt.Errorf("could not resolve registry %q", c.Registry)
	}

	if err = reg.LazyLoad(); err != nil {
		return fmt.Errorf("could not load plugin information from registry: %w", err)
	}

	var versions []registry.PluginVersion
	for _, p := range reg.Plugins {
		if p.Name == c.PluginName {
			versions = p.Versions
			break
		}
	}

	if len(versions) == 0 {
		return fmt.Errorf("plugin %s has no available versions to install", c.PluginName)
	}

	var targetVersion registry.PluginVersion
	if c.PluginVersion == "" {
		sort.Slice(versions, func(i, j int) bool {
			vi, vj := versions[i], versions[j]
			comp, _ := compareVersions(stringStringer(vi.Version), stringStringer(vj.Version))
			return comp > 0 // sort in reverse order
		})

		targetVersion = versions[0]
	} else {
		for _, v := range versions {
			comp, err := compareVersions(stringStringer(c.PluginVersion), stringStringer(v.Version))
			if err != nil {
				continue
			}

			if comp == 0 {
				targetVersion = v
				break
			}
		}
	}

	if targetVersion.Version == "" {
		return fmt.Errorf("could not find version %q in plugin registry %q", c.PluginVersion, c.Registry)
	}

	comp, err := compareVersions(bom.Version, stringStringer(targetVersion.Version))
	if err != nil {
		return fmt.Errorf("could not determine if %q was newer than %q", bom.Version, targetVersion.Version)
	}

	if comp >= 0 {
		return ErrPluginNewer
	}

	err = r.Run()
	if err != nil {
		return err
	}

	return i.Run()
}

type stringStringer string

func (s stringStringer) String() string {
	return string(s)
}
