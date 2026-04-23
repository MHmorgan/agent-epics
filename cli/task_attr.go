package cli

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/common"
	"github.com/MHmorgan/agent-epics/epic"
)

// registerAttrCommands registers attr:set, attr:get.
func registerAttrCommands(ctx context.Context) {
	registerAttrSetCmd(ctx)
	registerAttrGetCmd(ctx)
}

func registerAttrSetCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var epicArg, attr, value string
	cmd := app.SubCommand("attr:set", "Set an epic attribute")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.StringArg(&attr, "attr", "Attribute name")
	cmd.StringArg(&value, "value", "Attribute value")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()
		if err := epic.SetAttribute(ctx, q, attr, value); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}

func registerAttrGetCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var epicArg, attr string
	cmd := app.SubCommand("attr:get", "Get an epic attribute")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.StringArg(&attr, "attr", "Attribute name")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()
		value, err := epic.GetAttribute(ctx, q, attr)
		if err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(map[string]string{"value": value}))
		return nil
	})
}
