package install

import (
	"os"
	"path/filepath"
)

func (i *Command) installJAR() error {
	if err := i.installFromFileOrURL(i.PluginJAR, ".jar"); err != nil {
		return err
	}

	script := `#!/bin/sh
	
	java -jar ./export-plugin.jar
	`

	return os.WriteFile(filepath.Join(i.pluginHomeDir, i.pluginDir, "export-plugin"), []byte(script), 0o755)
}
