package update

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/blang/semver/v4"
	"github.com/gideaworx/terraform-exporter/install"
	"github.com/gideaworx/terraform-exporter/remove"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (c *Command) localUpdate() error {
	pluginFileName := c.PluginName
	if !strings.ContainsRune(pluginFileName, '/') {
		pluginFileName = fmt.Sprintf("./%s", pluginFileName)
	}

	i := &install.Command{
		LocalFile:  true,
		PluginName: c.PluginName,
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

	newPluginInfo, err := runner.LoadPluginInfo(pluginFileName, nil)
	if err != nil {
		return err
	}

	comp, err := compareVersions(bom.Version, newPluginInfo.Version)
	if err != nil {
		return fmt.Errorf("could not parse versions: %w", err)
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

func compareVersions(v1, v2 fmt.Stringer) (int, error) {
	s1, err := semver.ParseTolerant(v1.String())
	if err != nil {
		return 1, err
	}

	s2, err := semver.ParseTolerant(v2.String())
	if err != nil {
		return 1, err
	}

	return s1.Compare(s2), nil
}
