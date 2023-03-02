package install

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/gideaworx/terraform-exporter-plugin/go-plugin"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (i *Command) prepareNodeJSModule() (runner.BillOfMaterials, error) {
	normalizedName := strings.TrimPrefix(i.PluginNodeJS, "@")
	parts := strings.SplitN(normalizedName, "@", 2)
	bom := runner.BillOfMaterials{
		Name: parts[0],
		Type: runner.NodeJS,
	}

	if len(parts) > 1 {
		sv, err := semver.ParseTolerant(parts[1])
		if err == nil {
			bom.Version = plugin.FromSemver(sv)
		}
	}

	return bom, nil
}

func (i *Command) installNodePlugin() error {
	// this will strip the scope from a module name
	moduleName := path.Base(i.PluginNodeJS)

	var (
		npmPath string
		err     error
	)

	if _, err = exec.LookPath("node"); err != nil {
		return err
	}

	if npmPath, err = exec.LookPath("npm"); err != nil {
		return err
	}

	packageDir, err := filepath.Abs(filepath.Join(i.pluginHomeDir, moduleName))
	if err != nil {
		return err
	}

	dirInfo, err := os.Stat(packageDir)
	if err != nil {
		return err
	}

	if dirInfo.IsDir() {
		return fmt.Errorf("cannot install %s: %w", moduleName, ErrPluginAlreadyInstalled)
	}

	nmDir := filepath.Join(i.pluginHomeDir, moduleName, "node_modules")

	if err = os.MkdirAll(nmDir, 0o755); err != nil {
		return err
	}

	cmd := exec.Command(npmPath, "install", "--save", i.PluginNodeJS)
	cmd.Stdout = i.out
	cmd.Stderr = i.err
	cmd.Dir = filepath.Dir(nmDir)
	if err := cmd.Run(); err != nil {
		return err
	}

	return os.Symlink(filepath.Join(nmDir, ".bin", "export-plugin"), filepath.Join(nmDir, "..", "export-plugin"))
}
