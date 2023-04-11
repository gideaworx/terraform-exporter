package registry

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gideaworx/terraform-exporter-plugin-registry/registry"
	"github.com/gideaworx/terraform-exporter/runner"
	"gopkg.in/yaml.v3"
)

type PluginRegistry struct {
	URL          *url.URL          `yaml:",inline"`
	RegistryInfo []registry.Plugin `yaml:"-"`
}

func (r *PluginRegistry) Clone() *PluginRegistry {
	if r == nil || r.URL == nil {
		return nil
	}

	cloned := new(PluginRegistry)
	cloned.URL = r.URL.JoinPath("")
	cloned.RegistryInfo = make([]registry.Plugin, len(r.RegistryInfo))
	copy(cloned.RegistryInfo, r.RegistryInfo)

	return cloned
}

func (r *PluginRegistry) LazyLoad() error {
	if len(r.RegistryInfo) > 0 {
		return nil
	}
	hc := runner.GetHTTPClient()

	resp, err := hc.Get(r.URL.JoinPath("index.yaml").String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status '200 OK', got '%s'", resp.Status)
	}

	var fullRegistry registry.PluginRegistry
	if err = yaml.NewDecoder(resp.Body).Decode(&fullRegistry); err != nil {
		return err
	}

	r.RegistryInfo = fullRegistry.Plugins
	return nil
}

type PluginRegistries struct {
	r map[string]*PluginRegistry
	m *sync.RWMutex
}

type fileRegistryEntry struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type registryFile struct {
	Registries []fileRegistryEntry `yaml:"registries"`
}

const registryFileName = ".registries.yaml"

const defaultRegistryName = "default"
const defaultRegistryURLStr = "https://plugin-registry.gideaworx.io"

var defaultRegistryURL *url.URL

func init() {
	var err error
	defaultRegistryURL, err = url.Parse(defaultRegistryURLStr)
	if err != nil {
		panic(err)
	}
}

func LoadFromDisk() (*PluginRegistries, error) {
	pluginHome, err := runner.PluginHome()
	if err != nil {
		return nil, err
	}

	m := map[string]*PluginRegistry{
		defaultRegistryName: {
			URL: defaultRegistryURL,
		},
	}

	registries := &PluginRegistries{
		r: m,
		m: new(sync.RWMutex),
	}

	file, err := os.Open(filepath.Join(pluginHome, registryFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return registries, nil
		}
		return nil, err
	}
	defer file.Close()

	var installedRegistries registryFile
	if err = yaml.NewDecoder(file).Decode(&installedRegistries); err != nil {
		return nil, err
	}

	for _, reg := range installedRegistries.Registries {
		u, err := url.Parse(reg.URL)
		if err != nil {
			return nil, err
		}

		if err = registries.Add(reg.Name, u); err != nil {
			return nil, err
		}
	}

	return registries, nil
}

func (p *PluginRegistries) SaveToDisk() error {
	pluginHome, err := runner.PluginHome()
	if err != nil {
		return err
	}

	p.m.RLock()
	defer p.m.RUnlock()

	regFile := registryFile{
		Registries: make([]fileRegistryEntry, 0, len(p.r)),
	}
	for n, p := range p.r {
		if n == defaultRegistryName {
			continue
		}

		regFile.Registries = append(regFile.Registries, fileRegistryEntry{Name: n, URL: p.URL.String()})
	}

	contents, err := yaml.Marshal(regFile)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(pluginHome, registryFileName), contents, 0666)
}

func (p *PluginRegistries) GetAll() map[string]*PluginRegistry {
	p.m.RLock()
	defer p.m.RUnlock()

	c := make(map[string]*PluginRegistry, len(p.r))
	for n, r := range p.r {
		c[n] = r.Clone()
	}

	return c
}

func (p *PluginRegistries) Get(name string) *PluginRegistry {
	p.m.RLock()
	defer p.m.RUnlock()

	x := p.r[name]
	if x != nil {
		x = x.Clone()
	}

	return x
}

func (p *PluginRegistries) Add(name string, location *url.URL) error {
	if location == nil {
		return errors.New("location cannot be nil")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("a registry cannot be named with the empty string")
	}

	if name == "default" {
		return errors.New("you cannot override the default registry")
	}

	p.m.Lock()
	defer p.m.Unlock()
	if _, ok := p.r[name]; ok {
		return fmt.Errorf("a registry with name %s already exists", name)
	}

	p.r[name] = &PluginRegistry{
		URL: location,
	}
	return nil
}

func (p *PluginRegistries) Delete(name string) bool {
	p.m.Lock()
	defer p.m.Unlock()

	oldLen := len(p.r)
	delete(p.r, name)
	newLen := len(p.r)

	return newLen < oldLen
}
