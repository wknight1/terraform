package command

import (
	"fmt"
	"strings"

	"github.com/posener/complete"
)

type WorkspaceShowCommand struct {
	Meta
}

func (c *WorkspaceShowCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.extendedFlagSet("workspace show")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	workspace, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}
	c.Ui.Output(workspace)

	return 0
}

func (c *WorkspaceShowCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *WorkspaceShowCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceShowCommand) Help() string {
	helpText := `
Usage: terraform state current

  Show the name of the currently-selected named state.

  Typically a configuration as only one state associated with it, which is
  always named "default". If you have created additional named states then
  this command may return a different name.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceShowCommand) Synopsis() string {
	return "Show the name of the currently-selected state"
}
