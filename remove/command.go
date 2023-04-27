package remove

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gideaworx/terraform-exporter/runner"
)

type Command struct {
	PluginName     string    `arg:"" required:"true" help:"The name of the plugin to remove"`
	NonInteractive bool      `short:"f" default:"false" help:"Remove the plugin without asking first"`
	out            io.Writer `kong:"-"`
	in             io.Reader `kong:"-"`
}

func (c *Command) BeforeApply(stdout io.Writer, stdin io.Reader) error {
	c.out = stdout
	c.in = stdin

	return nil
}

func (c *Command) Run() error {
	pluginDir, err := runner.PluginDir(c.PluginName, false)
	if err != nil {
		return fmt.Errorf("could not get plugin directory: %w", err)
	}

	doDelete := c.NonInteractive
	if !doDelete {
		scanner := bufio.NewScanner(c.in)

		fmt.Fprintf(c.out, "Are you sure you want to delete plugin \x1b[1;96m%s\x1b[m? type 'y' or 'yes' (case insensitive): ", c.PluginName)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		doDelete = (strings.EqualFold("y", input) || strings.EqualFold("yes", input))

		if !doDelete {
			fmt.Fprintf(c.out, "\nYou answered %q, bailing out...\n\n", input)
			return nil
		}
	}

	return os.RemoveAll(pluginDir)
}
