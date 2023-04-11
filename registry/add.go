package registry

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/runner"
)

type AddRegistryCommand struct {
	Name string   `short:"n" help:"The name of the registry"`
	URL  *url.URL `short:"u" help:"The HTTPS URL hosting the registry's index.yaml. Do not include index.yaml in the URL"`
	r    *PluginRegistries
}

func (a *AddRegistryCommand) BeforeApply() error {
	var err error
	a.r, err = LoadFromDisk()
	return err
}

func (a *AddRegistryCommand) Run(ctx *kong.Context) error {
	testURL := a.URL.JoinPath("index.yaml")
	if testURL.Scheme != "https" {
		return fmt.Errorf("specified url %s has scheme %q, but it must have https", a.URL, a.URL.Scheme)
	}

	resp, err := runner.GetHTTPClient().Get(testURL.String())
	if err != nil {
		return fmt.Errorf("error validating registry URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(`expected http status "200 OK" from index.yaml, but got %q`, resp.Status)
	}

	if err := a.r.Add(a.Name, a.URL); err != nil {
		return err
	}

	return a.r.SaveToDisk()
}
