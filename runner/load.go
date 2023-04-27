package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/gideaworx/terraform-exporter-plugin/go-plugin"
	"github.com/hashicorp/go-hclog"
	goplug "github.com/hashicorp/go-plugin"
)

const PLUGIN_HOME = "TFE_PLUGIN_HOME"
const DEFAULT_PLUGIN_HOME = ".tf-exporter-plugins"

type PluginDefinition struct {
	info       plugin.PluginInformation
	p          plugin.ExportPlugin
	executable string
	client     *goplug.Client
}

var pluginMap map[int]goplug.PluginSet = map[int]goplug.PluginSet{
	int(plugin.RPCProtocol): {
		"plugin": &plugin.RPCExportPlugin{},
	},
	int(plugin.GRPCProtocol): {
		"plugin": &plugin.GRPCExportPlugin{},
	},
}

func (p PluginDefinition) PluginInfo() plugin.PluginInformation {
	return p.info
}

func (p PluginDefinition) Plugin() (plugin.ExportPlugin, error) {
	if p.p == nil {
		if p.client == nil {
			return nil, errors.New("cannot get plugin because the rpc client is unavailable")
		}

		rpcClient, err := p.client.Client()
		if err != nil {
			return nil, fmt.Errorf("could not connect to rpc client: %w", err)
		}

		raw, err := rpcClient.Dispense("plugin")
		if err != nil {
			return nil, fmt.Errorf("could not initialize plugin: %w", err)
		}

		plugin, ok := raw.(plugin.ExportPlugin)
		if !ok {
			return nil, fmt.Errorf("did not get the expected plugin implementation. expected an implementation of plugin.Plugin but got %T", plugin)
		}

		p.p = plugin
	}

	return p.p, nil
}

func (p PluginDefinition) Kill() {
	p.client.Kill()
}

func PluginHome() (string, error) {
	var err error
	home := os.Getenv(PLUGIN_HOME)
	if home == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not find user's home directory: %w", err)
		}

		home = filepath.Join(homedir, DEFAULT_PLUGIN_HOME)
	}

	if home, err = filepath.Abs(home); err != nil {
		return "", fmt.Errorf("could not get absolute path of plugins directory %s: %w", home, err)
	}

	info, err := os.Stat(home)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("could not read plugin directory %s: %w", home, err)
		}

		if err = os.MkdirAll(home, 0o777); err != nil {
			return "", err
		}
	}

	if info != nil && !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", home)
	}

	return home, nil
}

func PluginDir(pluginName string, create bool) (string, error) {
	home, err := PluginHome()
	if err != nil {
		return "", err
	}

	pluginDir := filepath.Join(home, pluginName)
	info, err := os.Stat(pluginDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("could not read plugin directory %s: %w", pluginDir, err)
		}

		if create {
			if err = os.MkdirAll(pluginDir, 0o777); err != nil {
				return "", err
			}
		}
	}

	if info != nil && !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", pluginDir)
	}

	return pluginDir, nil
}

func LoadPlugin(pluginName string, integrity *PluginIntegrity) (PluginDefinition, error) {
	home, err := PluginHome()
	if err != nil {
		return PluginDefinition{}, err
	}

	executable := filepath.Join(home, pluginName, "export-plugin")

	var checksum []byte
	var sc *goplug.SecureConfig
	if integrity == nil {
		bomFile := filepath.Join(home, pluginName, "export-plugin.bom")
		bomBytes, err := os.ReadFile(bomFile)
		if err != nil {
			return PluginDefinition{}, err
		}

		var bom BillOfMaterials
		if err = toml.Unmarshal(bomBytes, &bom); err != nil {
			return PluginDefinition{}, err
		}

		integrity = bom.Integrity
	}

	if integrity != nil {
		checksum, err = hex.DecodeString(integrity.Checksum)
		if err != nil {
			return PluginDefinition{}, err
		}
		sc = &goplug.SecureConfig{
			Hash:     sha256.New(),
			Checksum: checksum,
		}
	}

	client := goplug.NewClient(&goplug.ClientConfig{
		HandshakeConfig:  plugin.HandshakeConfig,
		VersionedPlugins: pluginMap,
		Cmd:              exec.Command(executable),
		SecureConfig:     sc,
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   "plugin",
			Level:  hclog.Debug,
			Output: os.Stdout,
		}),
	})

	c, err := client.Client()
	if err != nil {
		return PluginDefinition{}, fmt.Errorf("error creating plugin client: %w", err)
	}

	raw, err := c.Dispense("plugin")
	if err != nil {
		return PluginDefinition{}, fmt.Errorf("error loading plugin: %w", err)
	}

	ep, ok := raw.(plugin.ExportPlugin)
	if !ok {
		return PluginDefinition{}, errors.New("client did not dispense an ExportPlugin")
	}

	info, err := ep.Info()
	if err != nil {
		return PluginDefinition{}, err
	}

	return PluginDefinition{
		info:       info,
		p:          ep,
		executable: executable,
		client:     client,
	}, nil
}

func LoadPluginInfo(pluginName string, integrity *PluginIntegrity) (plugin.PluginInformation, error) {
	pd, err := LoadPlugin(pluginName, integrity)
	if err != nil {
		return plugin.PluginInformation{}, err
	}
	defer pd.client.Kill()

	return pd.PluginInfo(), nil
}
