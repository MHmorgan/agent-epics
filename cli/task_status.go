package cli

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/common"
	"github.com/MHmorgan/agent-epics/epic"
)

// registerStatusCommands registers task:start, task:block, task:unblock, task:done, task:abandon.
func registerStatusCommands(ctx context.Context) {
	registerStartCmd(ctx)
	registerBlockCmd(ctx)
	registerUnblockCmd(ctx)
	registerDoneCmd(ctx)
	registerAbandonCmd(ctx)
}

func registerStartCmd(ctx context.Context) {
	var rawID string
	cmd := app.SubCommand("task:start", "Start working on a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		return transitionTask(ctx, rawID, epic.StatusActive, "")
	})
}

func registerBlockCmd(ctx context.Context) {
	var rawID, reason string
	cmd := app.SubCommand("task:block", "Block a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&reason, "reason", "Reason for blocking")
	cmd.Run(func() error {
		return transitionTask(ctx, rawID, epic.StatusBlocked, reason)
	})
}

func registerUnblockCmd(ctx context.Context) {
	var rawID string
	cmd := app.SubCommand("task:unblock", "Unblock a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		return transitionTask(ctx, rawID, epic.StatusActive, "")
	})
}

func registerDoneCmd(ctx context.Context) {
	var rawID string
	cmd := app.SubCommand("task:done", "Complete a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		return transitionTask(ctx, rawID, epic.StatusDone, "")
	})
}

func registerAbandonCmd(ctx context.Context) {
	var rawID, reason string
	cmd := app.SubCommand("task:abandon", "Abandon a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&reason, "reason", "Reason for abandoning")
	cmd.Run(func() error {
		return transitionTask(ctx, rawID, epic.StatusAbandoned, reason)
	})
}

// transitionTask parses the task ID, opens the epic, and performs the status transition.
func transitionTask(ctx context.Context, rawID string, to epic.Status, reason string) error {
	cfg := common.GetConfig(ctx)
	taskID, err := epic.ParseTaskID(rawID)
	if err != nil {
		return jsonError(err)
	}
	conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
	if err != nil {
		return jsonError(err)
	}
	defer conn.Close()
	if err := epic.TransitionStatus(ctx, conn, q, taskID, to, reason); err != nil {
		return jsonError(err)
	}
	fmt.Println(jsonSuccess(nil))
	return nil
}
