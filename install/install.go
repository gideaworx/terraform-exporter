package install

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gideaworx/terraform-exporter/runner"
)

func (i *Command) install() error {
	if i.PluginNative != "" {
		return i.installFromFileOrURL(i.PluginNative, "")
	}

	if i.PluginJAR != "" {
		return i.installJAR()
	}

	if i.PluginNodeJS != "" {
		return i.installNodePlugin()
	}

	if i.PluginPython != "" {
		return i.installPythonPlugin()
	}

	return errors.New("exactly one of --plugin-native, --plugin-jar, --plugin-nodejs, or --plugin-python must be set")
}

func (i *Command) installFromFileOrURL(loc string, extension string) error {
	var reader io.ReadCloser
	var err error
	if strings.HasPrefix(strings.ToLower(loc), "https://") {
		resp, err := runner.GetHTTPClient().Get(loc)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected HTTP 200 OK from %q but got %s", loc, resp.Status)
		}

		reader = resp.Body
	} else {
		reader, err = os.Open(loc)
		if err != nil {
			return err
		}
	}

	defer reader.Close()

	decompressed, err := i.getDecompressedReader(reader)
	if err != nil {
		return err
	}
	defer decompressed.Close()

	destFile, err := os.OpenFile(filepath.Join(i.pluginHomeDir, i.pluginDir, fmt.Sprintf("export-plugin%s", extension)), os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, decompressed)
	return err
}
