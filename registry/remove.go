package registry

import (
	"fmt"

	"github.com/alecthomas/kong"
)

type RemoveRegistryCommand struct {
	Name string `arg:"" help:"The name of the registry to remove"`
	r    *PluginRegistries
}

func (r *RemoveRegistryCommand) BeforeApply() error {
	var err error
	r.r, err = LoadFromDisk()
	return err
}

func (r *RemoveRegistryCommand) Run(ctx *kong.Context) error {
	if reg := r.r.Get(r.Name); reg == nil {
		return fmt.Errorf("registry %q not installed", r.Name)
	}

	if r.r.Delete(r.Name) {
		return r.r.SaveToDisk()
	}

	return nil
}
