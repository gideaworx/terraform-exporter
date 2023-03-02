package help

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/gideaworx/terraform-exporter/runner"
)

type Command struct {
	CommandName string `arg:"" help:"The name of the command to use for export"`
}

func (c *Command) Run(ctx *kong.Context, output io.Writer) error {
	matching, err := runner.FindPluginsForCommand(c.CommandName)
	if err != nil {
		return err
	}

	if len(matching) == 0 {
		return fmt.Errorf("no installed plugins provide command %q", c.CommandName)
	}

	if len(matching) > 1 {
		options := []string{}
		for _, m := range matching {
			options = append(options, fmt.Sprintf("%s/%s", m[0], m[1]))
		}

		return fmt.Errorf("multiple plugins provide command %q. Valid choices are %q", c.CommandName, strings.Join(options, ", "))
	}

	pDef, err := runner.LoadPlugin(matching[0][0], nil)
	if err != nil {
		return err
	}
	defer pDef.Kill()

	impl, err := pDef.Plugin()
	if err != nil {
		return err
	}

	helpTxt, err := impl.Help(matching[0][1])
	if err != nil {
		return err
	}

	if output == nil {
		output = ctx.Stdout
	}

	helpTxt = strings.ReplaceAll(helpTxt, "\n", "\n\t")
	fmt.Fprintf(output, "Command %s/%s Help\n\n", matching[0][0], matching[0][1])
	fmt.Fprintln(output, helpTxt)

	return nil
}
