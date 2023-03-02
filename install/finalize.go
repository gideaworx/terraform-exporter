package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/gideaworx/terraform-exporter-plugin/go-plugin"
	"github.com/gideaworx/terraform-exporter/runner"
)

func (i *Command) finalize() error {
	switch {
	case i.PluginNative != "":
		return i.finalizeFromInfo(runner.Native)
	case i.PluginJAR != "":
		return i.finalizeFromInfo(runner.JVM)
	case i.PluginNodeJS != "":
		return i.finalizeFromInfo(runner.NodeJS)
	case i.PluginPython != "":
		return i.finalizeFromInfo(runner.Python)
	default:
		return errors.New("unrecognized plugin type")
	}
}

func (i *Command) calculateIntegrity() (*runner.PluginIntegrity, error) {
	pluginName := filepath.Join(i.pluginHomeDir, i.pluginDir, "export-plugin")

	pluginFile, err := os.Open(pluginName)
	if err != nil {
		return nil, err
	}
	defer pluginFile.Close()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, pluginFile); err != nil {
		return nil, err
	}

	checksum := hasher.Sum(nil)

	return &runner.PluginIntegrity{
		Checksum:  hex.EncodeToString(checksum),
		Algorithm: "sha256",
	}, nil
}

func (i *Command) finalizeFromInfo(pluginType runner.PluginType) error {
	integrity, err := i.calculateIntegrity()
	if err != nil {
		return err
	}

	info, err := runner.LoadPluginInfo(i.pluginDir, integrity)
	if err != nil {
		return err
	}

	bomFile := filepath.Join(i.pluginHomeDir, i.pluginDir, "export-plugin.bom")
	bomBytes, err := os.ReadFile(bomFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var bom runner.BillOfMaterials
	if bomBytes != nil {
		if err = toml.Unmarshal(bomBytes, &bom); err != nil {
			return err
		}
	}

	if bom.Name == "" {
		bom.Name = i.pluginDir
	}

	bom.Type = pluginType
	bom.Version = info.Version
	bom.Provides = append([]plugin.CommandInfo{}, info.Provides...)
	bom.Integrity = integrity

	file, err := os.Create(bomFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return toml.NewEncoder(file).Encode(bom)
}
