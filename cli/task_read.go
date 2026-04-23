package cli

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/common"
	"github.com/MHmorgan/agent-epics/epic"
)

// registerTaskReadCommands registers task, task:list, task:records, task:next.
func registerTaskReadCommands(ctx context.Context) {
	registerTaskCmd(ctx)
	registerTaskListCmd(ctx)
	registerTaskRecordsCmd(ctx)
	registerTaskNextCmd(ctx)
}

func registerTaskCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID string
	cmd := app.SubCommand("task", "Get a task")
	cmd.StringArg(&rawID, "id", "Task ID")
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

		task, err := epic.GetTask(ctx, q, taskID)
		if err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(task))
		return nil
	})
}

func registerTaskListCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var epicArg string
	var showAll bool
	var parentFlag string

	cmd := app.SubCommand("task:list", "List tasks in an epic")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.BoolFlag(&showAll, "all", "Include terminal tasks")
	cmd.StringFlag(&parentFlag, "parent", "Filter to children of this ID")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()

		var parentID epic.TaskID
		if parentFlag != "" {
			parentID, err = epic.ParseTaskID(parentFlag)
			if err != nil {
				return jsonError(err)
			}
		}

		tasks, err := epic.ListTasks(ctx, q, parentID, showAll)
		if err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(tasks))
		return nil
	})
}

func registerTaskRecordsCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID string
	var selfOnly bool

	cmd := app.SubCommand("task:records", "Get records")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.BoolFlag(&selfOnly, "self", "Exact match only instead of subtree")
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

		records, err := epic.GetRecords(ctx, q, taskID, selfOnly)
		if err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(records))
		return nil
	})
}

func registerTaskNextCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var epicArg string
	cmd := app.SubCommand("task:next", "Get next ready task")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()

		task, err := epic.NextTask(ctx, conn, q, epicArg)
		if err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(task))
		return nil
	})
}
