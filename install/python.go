package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/gideaworx/terraform-exporter-plugin/go-plugin"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (i *Command) preparePythonPackage() (runner.BillOfMaterials, error) {
	parts := strings.SplitN(i.PluginPython, "==", 2)

	bom := runner.BillOfMaterials{
		Name: parts[0],
	}

	if len(parts) > 1 {
		sv, err := semver.ParseTolerant(parts[1])
		if err == nil {
			bom.Version = plugin.FromSemver(sv)
		}
	}

	return bom, nil
}

func (i *Command) installPythonPlugin() error {
	var (
		pythonPath string
		err        error
	)

	if pythonPath, err = exec.LookPath("python3"); err != nil {
		if pythonPath, err = exec.LookPath("python"); err != nil {
			return errors.New(`could not find "python" or "python3" on the system PATH`)
		}
	}

	packageDir, err := filepath.Abs(filepath.Join(i.pluginHomeDir, i.PluginPython))
	if err != nil {
		return err
	}

	if err = os.MkdirAll(packageDir, 0o755); err != nil {
		return err
	}

	ensurePipCmd := exec.Command(pythonPath, "-m", "ensurepip")
	ensurePipCmd.Stdout = i.out
	ensurePipCmd.Stderr = i.err
	if err = ensurePipCmd.Run(); err != nil {
		return fmt.Errorf("could not ensure pip is installed: %w", err)
	}

	pipInstaller := exec.Command(pythonPath, "-m", "pip", "install", "--target", packageDir, i.PluginPython)
	pipInstaller.Stdout = i.out
	pipInstaller.Stderr = i.err
	if err := pipInstaller.Run(); err != nil {
		return err
	}

	return os.Symlink(filepath.Join(packageDir, "bin", "export-plugin"), filepath.Join(packageDir, "export-plugin"))
}
