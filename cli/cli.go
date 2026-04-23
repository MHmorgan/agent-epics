package cli

import (
	"context"
	"errors"

	"github.com/Minimal-Viable-Software/cli-go"
)

var app = cli.NewApplication()

// Run the application
func Run(ctx context.Context, args []string) error {
	registerHumanCommands(ctx)
	registerTaskWriteCommands(ctx)
	registerTaskReadCommands(ctx)
	registerStructureCommands(ctx)
	registerStatusCommands(ctx)
	registerAttrCommands(ctx)

	err := app.Parse(args)
	if errors.Is(err, cli.ErrHelp) {
		return nil
	}
	return err
}
