package install

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (i *Command) prepare() error {
	var bom runner.BillOfMaterials
	var err error

	switch {
	case i.PluginNative != "":
		bom, err = i.prepareFromFileOrURL(i.PluginNative, runner.Native)
	case i.PluginJAR != "":
		bom, err = i.prepareFromFileOrURL(i.PluginJAR, runner.JVM)
	case i.PluginNodeJS != "":
		bom, err = i.prepareNodeJSModule()
	case i.PluginPython != "":
		bom, err = i.preparePythonPackage()
	}

	if err != nil {
		return err
	}

	i.pluginDir = normalizeDirName(bom.Name)
	dir := filepath.Join(i.pluginHomeDir, i.pluginDir)
	info, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("there is already a plugin called %q installed. Use the update command instead", bom.Name)
	}

	if err := os.MkdirAll(filepath.Join(i.pluginHomeDir, i.pluginDir), 0o755); err != nil {
		return err
	}

	bomFile, err := os.Create(filepath.Join(i.pluginHomeDir, i.pluginDir, "export-plugin.bom"))
	if err != nil {
		return err
	}
	defer bomFile.Close()

	return toml.NewEncoder(bomFile).Encode(bom)
}

func (i *Command) prepareFromFileOrURL(loc string, pType runner.PluginType) (runner.BillOfMaterials, error) {
	pluginFile := strings.ReplaceAll(loc, "\\", "/")
	if strings.HasPrefix(strings.ToLower(loc), "https://") {
		u, err := url.Parse(loc)
		if err != nil {
			return runner.BillOfMaterials{}, err
		}
		pluginFile = u.EscapedPath()
	}

	name := path.Base(pluginFile)
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, "."+ext)

	return runner.BillOfMaterials{
		Name: name,
		Type: pType,
	}, nil
}

func normalizeDirName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsPrint(r):
			switch r {
			case '@', '/', '_', '.':
				return '-'
			default:
				return unicode.ToLower(r)
			}
		}

		return -1
	}, s)
}
