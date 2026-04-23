package cli

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/common"
	"github.com/MHmorgan/agent-epics/epic"
)

// registerStructureCommands registers task:split, task:unsplit, task:after, task:unafter.
func registerStructureCommands(ctx context.Context) {
	registerSplitCmd(ctx)
	registerUnsplitCmd(ctx)
	registerAfterCmd(ctx)
	registerUnafterCmd(ctx)
}

func registerSplitCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID string
	cmd := app.SubCommand("task:split", "Split a task into subtasks")
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
		if err := epic.SplitTask(ctx, conn, q, taskID); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}

func registerUnsplitCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID string
	cmd := app.SubCommand("task:unsplit", "Unsplit a task")
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
		if err := epic.UnsplitTask(ctx, conn, q, taskID); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}

func registerAfterCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID, rawPred string
	cmd := app.SubCommand("task:after", "Add a dependency (task after predecessor)")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&rawPred, "pred", "Predecessor task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			return jsonError(err)
		}
		predID, err := epic.ParseTaskID(rawPred)
		if err != nil {
			return jsonError(err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()
		if err := epic.AddDependency(ctx, q, taskID, predID); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}

func registerUnafterCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID, rawPred string
	cmd := app.SubCommand("task:unafter", "Remove a dependency")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&rawPred, "pred", "Predecessor task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			return jsonError(err)
		}
		predID, err := epic.ParseTaskID(rawPred)
		if err != nil {
			return jsonError(err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return jsonError(err)
		}
		defer conn.Close()
		if err := epic.RemoveDependency(ctx, q, taskID, predID); err != nil {
			return jsonError(err)
		}
		fmt.Println(jsonSuccess(nil))
		return nil
	})
}
