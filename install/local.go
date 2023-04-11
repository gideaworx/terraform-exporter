package install

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (i *Command) localInstall() error {
	path, err := filepath.Abs(i.PluginName)
	if err != nil {
		return err
	}

	pluginDir := filepath.Join(i.pluginHomeDir, filepath.Base(path))
	if err := os.MkdirAll(pluginDir, 0777); err != nil {
		return err
	}

	targetFile, err := os.OpenFile(filepath.Join(pluginDir, "export-plugin"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(targetFile, hasher)

	contents, err := os.Open(path)
	if err != nil {
		return err
	}
	defer contents.Close()

	if _, err = io.Copy(writer, contents); err != nil {
		return err
	}
	targetFile.Close()

	integrity := &runner.PluginIntegrity{
		Checksum:  hex.EncodeToString(hasher.Sum(nil)),
		Algorithm: "sha256",
	}

	info, err := runner.LoadPluginInfo(filepath.Base(path), integrity)
	if err != nil {
		return err
	}

	bom := runner.BillOfMaterials{
		Name: filepath.Base(path),
		Type: runner.Native,
		Source: runner.PluginSource{
			Type: "local-file",
			Name: path,
		},
		Integrity: integrity,
		Version:   info.Version,
		Provides:  info.Provides,
	}

	bomFile, err := os.Create(filepath.Join(pluginDir, "export-plugin.bom"))
	if err != nil {
		return err
	}
	defer bomFile.Close()

	return toml.NewEncoder(bomFile).Encode(bom)
}
