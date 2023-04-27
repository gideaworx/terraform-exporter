package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/blang/semver/v4"
	"github.com/gideaworx/terraform-exporter-plugin-registry/registry"
	"github.com/gideaworx/terraform-exporter-plugin/go-plugin"
	localreg "github.com/gideaworx/terraform-exporter/registry"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (i *Command) registryInstall() error {
	reg := i.r.Get(i.Registry)
	if reg == nil {
		return fmt.Errorf("no plugin %q found", i.Registry)
	}

	if err := reg.LazyLoad(); err != nil {
		return err
	}

	var plugin registry.Plugin
	var version registry.PluginVersion
	found := false
	for _, p := range reg.Plugins {
		if p.Name == i.PluginName {
			plugin = p
			if strings.EqualFold(strings.TrimSpace(i.PluginVersion), "") {
				found = true
				break
			}

			for _, v := range p.Versions {
				if v.Version == i.PluginVersion {
					version = v
					found = true
					break
				}
			}
		}
	}

	if !found {
		return fmt.Errorf("could not find version %q of plugin %q in registry %q", i.PluginVersion, i.PluginName, i.Registry)
	}

	if strings.EqualFold(strings.TrimSpace(i.PluginVersion), "") {
		sort.Slice(plugin.Versions, func(i, j int) bool {
			si, _ := semver.ParseTolerant(plugin.Versions[i].Version)
			sj, _ := semver.ParseTolerant(plugin.Versions[j].Version)

			return si.GT(sj)
		})

		version = plugin.Versions[0]
	}

	exe := i.getPluginExecutable(version)
	if exe.Locator == "" {
		return fmt.Errorf("plugin %s, version %s is not compatible with architecture %s/%s", plugin.Name, version.Version, runtime.GOOS, runtime.GOARCH)
	}

	pluginDir := filepath.Join(i.pluginHomeDir, i.PluginName)
	if err := os.MkdirAll(pluginDir, 0777); err != nil {
		return err
	}

	var installer func(string, registry.PluginExecutable, *localreg.PluginRegistry, string) (runner.BillOfMaterials, error)

	switch exe.Type {
	case registry.Native:
		installer = i.installNative
	case registry.NodeJS:
		installer = i.installNPM
	case registry.Python:
		installer = i.installPyPI
	// case registry.Java:
	// 	return i.installMaven(exe, reg, version.Version)
	default:
		return fmt.Errorf("unknown type %s", exe.Type)
	}

	bom, err := installer(pluginDir, exe, reg, version.Version)
	if err != nil {
		return err
	}

	bomFile, err := os.Create(filepath.Join(pluginDir, "export-plugin.bom"))
	if err != nil {
		return err
	}
	defer bomFile.Close()

	return toml.NewEncoder(bomFile).Encode(bom)
}

func (i *Command) getPluginExecutable(v registry.PluginVersion) registry.PluginExecutable {
	targetArch := registry.TargetArchitecture(fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	if exe, ok := v.DownloadInfo[targetArch]; ok {
		return exe
	}

	if maExe, ok := v.DownloadInfo[registry.MultiArch]; ok {
		return maExe
	}

	// if we're on darwin/arm64, we can try darwin/amd64 and let Rosetta take the wheel
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return v.DownloadInfo[registry.DarwinAmd64]
	}

	return registry.PluginExecutable{}
}

func (i *Command) installNative(pluginDir string, exe registry.PluginExecutable, reg *localreg.PluginRegistry, version string) (runner.BillOfMaterials, error) {
	targetFile, err := os.OpenFile(filepath.Join(pluginDir, "export-plugin"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return runner.BillOfMaterials{}, err
	}
	defer targetFile.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(targetFile, hasher)

	resp, err := runner.GetHTTPClient().Get(exe.Locator)
	if err != nil {
		return runner.BillOfMaterials{}, err
	}
	defer resp.Body.Close()

	if _, err = io.Copy(writer, resp.Body); err != nil {
		return runner.BillOfMaterials{}, err
	}
	targetFile.Close()

	calculatedSHASum := hex.EncodeToString(hasher.Sum(nil))
	if exe.Info.Checksum != "" && exe.Info.Checksum != calculatedSHASum {
		return runner.BillOfMaterials{}, fmt.Errorf("calculated sha256 checksum %q does not match provided checksum %q", calculatedSHASum, exe.Info.Checksum)
	}

	integrity := &runner.PluginIntegrity{
		Checksum:  calculatedSHASum,
		Algorithm: "sha256",
	}

	info, err := runner.LoadPluginInfo(i.PluginName, integrity)
	if err != nil {
		return runner.BillOfMaterials{}, err
	}

	bom := runner.BillOfMaterials{
		Name:    i.PluginName,
		Version: plugin.FromString(version),
		Type:    runner.Native,
		Source: runner.PluginSource{
			Type: "registry",
			Name: i.Registry,
			URL:  reg.URL.String(),
		},
		Integrity: integrity,
		Provides:  info.Provides,
	}

	return bom, nil
}

func (i *Command) installPyPI(pluginDir string, exe registry.PluginExecutable, reg *localreg.PluginRegistry, version string) (runner.BillOfMaterials, error) {
	var (
		pythonPath string
		err        error
	)

	if pythonPath, err = exec.LookPath("python3"); err != nil {
		if pythonPath, err = exec.LookPath("python"); err != nil {
			return runner.BillOfMaterials{}, errors.New(`could not find "python" or "python3" on the system PATH`)
		}
	}

	ensurePipCmd := exec.Command(pythonPath, "-m", "ensurepip")
	ensurePipCmd.Stdout = i.out
	ensurePipCmd.Stderr = i.err
	if err = ensurePipCmd.Run(); err != nil {
		return runner.BillOfMaterials{}, fmt.Errorf("could not ensure pip is installed: %w", err)
	}

	pipArgs := append([]string{"-m", "pip", "install", "--target", pluginDir, fmt.Sprintf("%s==%s", exe.Locator, version)}, exe.Info.ExtraArgs...)

	pipInstaller := exec.Command(pythonPath, pipArgs...)
	pipInstaller.Stdout = i.out
	pipInstaller.Stderr = i.err
	if err := pipInstaller.Run(); err != nil {
		return runner.BillOfMaterials{}, err
	}

	if err = os.Symlink(filepath.Join(pluginDir, "bin", "export-plugin"), filepath.Join(pluginDir, "export-plugin")); err != nil {
		return runner.BillOfMaterials{}, err
	}

	hasher := sha256.New()
	contents, err := os.ReadFile(filepath.Join(pluginDir, "bin", "export-plugin"))
	if err != nil {
		return runner.BillOfMaterials{}, err
	}

	checksum := hasher.Sum(contents)
	integrity := &runner.PluginIntegrity{
		Checksum:  hex.EncodeToString(checksum),
		Algorithm: "sha256",
	}

	info, err := runner.LoadPluginInfo(i.PluginName, integrity)
	if err != nil {
		return runner.BillOfMaterials{}, err
	}

	return runner.BillOfMaterials{
		Name:    i.PluginName,
		Version: info.Version,
		Type:    runner.Python,
		Source: runner.PluginSource{
			Type: "registry",
			Name: i.Registry,
			URL:  reg.URL.String(),
		},
		Integrity: integrity,
		Provides:  info.Provides,
	}, nil
}

func (i *Command) installNPM(pluginDir string, exe registry.PluginExecutable, reg *localreg.PluginRegistry, version string) (runner.BillOfMaterials, error) {
	var (
		npmPath string
		bom     runner.BillOfMaterials
		err     error
	)

	if _, err = exec.LookPath("node"); err != nil {
		return bom, err
	}

	if npmPath, err = exec.LookPath("npm"); err != nil {
		return bom, err
	}

	nmDir := filepath.Join(pluginDir, "node_modules")
	if err = os.MkdirAll(nmDir, 0o755); err != nil {
		return bom, err
	}

	npmArgs := append([]string{"install", "--save", fmt.Sprintf("%s@%s", exe.Locator, version)}, exe.Info.ExtraArgs...)
	cmd := exec.Command(npmPath, npmArgs...)
	cmd.Stdout = i.out
	cmd.Stderr = i.err
	cmd.Dir = filepath.Dir(nmDir)
	if err := cmd.Run(); err != nil {
		return bom, err
	}

	if err = os.Symlink(filepath.Join(nmDir, ".bin", "export-plugin"), filepath.Join(pluginDir, "export-plugin")); err != nil {
		return bom, err
	}

	hasher := sha256.New()
	contents, err := os.ReadFile(filepath.Join(nmDir, ".bin", "export-plugin"))
	if err != nil {
		return runner.BillOfMaterials{}, err
	}

	checksum := hasher.Sum(contents)
	integrity := &runner.PluginIntegrity{
		Checksum:  hex.EncodeToString(checksum),
		Algorithm: "sha256",
	}

	info, err := runner.LoadPluginInfo(i.PluginName, integrity)
	if err != nil {
		return runner.BillOfMaterials{}, err
	}

	return runner.BillOfMaterials{
		Name:    i.PluginName,
		Version: info.Version,
		Type:    runner.NodeJS,
		Source: runner.PluginSource{
			Type: "registry",
			Name: i.Registry,
			URL:  reg.URL.String(),
		},
		Integrity: integrity,
		Provides:  info.Provides,
	}, nil
}
