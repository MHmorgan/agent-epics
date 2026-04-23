package cli

import (
	"context"
	"fmt"

	clilib "github.com/Minimal-Viable-Software/cli-go"

	"github.com/MHmorgan/agent-epics/epic"
)

// RegisterAttrCommands registers attr:set, attr:get.
func RegisterAttrCommands(app *clilib.Application, epicsDir string) {
	registerAttrSetCmd(app, epicsDir)
	registerAttrGetCmd(app, epicsDir)
}

func registerAttrSetCmd(app *clilib.Application, epicsDir string) {
	var epicArg, attr, value string
	cmd := app.SubCommand("attr:set", "Set an epic attribute")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.StringArg(&attr, "attr", "Attribute name")
	cmd.StringArg(&value, "value", "Attribute value")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, epicsDir)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		defer conn.Close()
		ctx := context.Background()
		if err := epic.SetAttribute(ctx, q, attr, value); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}

func registerAttrGetCmd(app *clilib.Application, epicsDir string) {
	var epicArg, attr string
	cmd := app.SubCommand("attr:get", "Get an epic attribute")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.StringArg(&attr, "attr", "Attribute name")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, epicsDir)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		defer conn.Close()
		ctx := context.Background()
		value, err := epic.GetAttribute(ctx, q, attr)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(map[string]string{"value": value}))
		return nil
	})
}
