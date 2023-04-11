package runner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/gideaworx/terraform-exporter-plugin/go-plugin"
)

type PluginType string

const (
	Native PluginType = "native"
	NodeJS PluginType = "nodejs"
	Python PluginType = "python"
	Java   PluginType = "java"
)

type PluginIntegrity struct {
	Checksum  string `toml:"checksum"`
	Algorithm string `toml:"algorithm"`
}

type PluginSource struct {
	Type string `toml:"type"`
	Name string `toml:"name"`
	URL  string `toml:"url,omitempty"`
}

type BillOfMaterials struct {
	Name      string               `toml:"name"`
	Type      PluginType           `toml:"type"`
	Source    PluginSource         `toml:"source"`
	Version   plugin.Version       `toml:"version,omitempty"`
	Integrity *PluginIntegrity     `toml:"integrity,omitempty"`
	Provides  []plugin.CommandInfo `toml:"provides,omitempty"`
}

func FindPluginsForCommand(cmdName string) ([][2]string, error) {
	boms, err := LoadInstalledBOMs()
	if err != nil {
		return nil, err
	}

	if strings.Contains(cmdName, "/") {
		parts := strings.Split(cmdName, "/")
		for i := 0; i < len(boms); i++ {
			if parts[0] == boms[i].Name {
				boms = []BillOfMaterials{boms[i]}
				break
			}
		}
	}

	matching := [][2]string{}
	cmdName = cmdName[strings.Index(cmdName, "/")+1:]
	for _, bom := range boms {
		for _, c := range bom.Provides {
			if c.Name == cmdName {
				matching = append(matching, [2]string{bom.Name, c.Name})
				break
			}
		}
	}

	return matching, nil
}

func LoadInstalledBOMs() ([]BillOfMaterials, error) {
	home, err := PluginHome()
	if err != nil {
		return nil, err
	}

	pattern := filepath.Join(home, "*", "export-plugin.bom")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	boms := make([]BillOfMaterials, 0, len(matches))
	lock := &sync.Mutex{}
	errs := []error{}

	wg := &sync.WaitGroup{}
	wg.Add(len(matches))
	for _, match := range matches {
		go func(bomFile string) {
			defer wg.Done()
			bomBytes, err := os.ReadFile(bomFile)
			if err != nil {
				errs = append(errs, err)
				return
			}

			var bom BillOfMaterials
			if err = toml.Unmarshal(bomBytes, &bom); err != nil {
				errs = append(errs, err)
				return
			}

			lock.Lock()
			boms = append(boms, bom)
			lock.Unlock()
		}(match)
	}
	wg.Wait()

	if len(errs) > 0 {
		msg := "the following errors occurred loading bills of material: "
		errstrs := []string{}
		for _, e := range errs {
			errstrs = append(errstrs, e.Error())
		}

		msg += strings.Join(errstrs, ", ")

		return nil, errors.New(msg)
	}

	return boms, nil
}
