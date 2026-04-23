package cli

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/common"
	"github.com/MHmorgan/agent-epics/epic"
)

// registerTaskWriteCommands registers task:new-epic, task:add-child, task:set,
// task:context:set, task:record.
func registerTaskWriteCommands(ctx context.Context) {
	registerNewEpicCmd(ctx)
	registerAddChildCmd(ctx)
	registerSetCmd(ctx)
	registerContextSetCmd(ctx)
	registerRecordCmd(ctx)
}

func registerNewEpicCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var epicID string
	cmd := app.SubCommand("task:new-epic", "Create a new epic")
	cmd.StringArg(&epicID, "epic", "Epic ID")
	cmd.Run(func() error {
		if err := epic.NewEpic(epicID, cfg.EpicsDir); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(map[string]string{"id": epicID}))
		return nil
	})
}

func registerAddChildCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var parent string
	cmd := app.SubCommand("task:add-child", "Add child to branch")
	cmd.StringArg(&parent, "parent", "Parent task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(parent)
		if err != nil {
			return jsonError(err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()

		childID, err := epic.AddChild(ctx, conn, q, taskID)
		if err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(map[string]string{"id": childID.String()}))
		return nil
	})
}

func registerSetCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID, markdown string
	cmd := app.SubCommand("task:set", "Set task body")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&markdown, "markdown", "Body text")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			return jsonError(err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()

		if err := epic.SetTaskBody(ctx, conn, q, taskID, markdown); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}

func registerContextSetCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID, markdown string
	cmd := app.SubCommand("task:context:set", "Set task context")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&markdown, "markdown", "Context text")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			return jsonError(err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()

		if err := epic.SetTaskContext(ctx, conn, q, taskID, markdown); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}

func registerRecordCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID, text string
	cmd := app.SubCommand("task:record", "Append agent record")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&text, "text", "Record text")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			return jsonError(err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()

		if err := epic.AddRecord(ctx, q, taskID, text); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}
