package cli

import (
	"context"
	"fmt"

	clilib "github.com/Minimal-Viable-Software/cli-go"

	"github.com/MHmorgan/agent-epics/epic"
)

// RegisterStatusCommands registers task:start, task:block, task:unblock, task:done, task:abandon.
func RegisterStatusCommands(app *clilib.Application, epicsDir string) {
	registerStartCmd(app, epicsDir)
	registerBlockCmd(app, epicsDir)
	registerUnblockCmd(app, epicsDir)
	registerDoneCmd(app, epicsDir)
	registerAbandonCmd(app, epicsDir)
}

func registerStartCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	cmd := app.SubCommand("task:start", "Start working on a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		return transitionTask(rawID, epicsDir, epic.StatusActive, "")
	})
}

func registerBlockCmd(app *clilib.Application, epicsDir string) {
	var rawID, reason string
	cmd := app.SubCommand("task:block", "Block a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&reason, "reason", "Reason for blocking")
	cmd.Run(func() error {
		return transitionTask(rawID, epicsDir, epic.StatusBlocked, reason)
	})
}

func registerUnblockCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	cmd := app.SubCommand("task:unblock", "Unblock a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		return transitionTask(rawID, epicsDir, epic.StatusActive, "")
	})
}

func registerDoneCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	cmd := app.SubCommand("task:done", "Complete a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		return transitionTask(rawID, epicsDir, epic.StatusDone, "")
	})
}

func registerAbandonCmd(app *clilib.Application, epicsDir string) {
	var rawID, reason string
	cmd := app.SubCommand("task:abandon", "Abandon a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&reason, "reason", "Reason for abandoning")
	cmd.Run(func() error {
		return transitionTask(rawID, epicsDir, epic.StatusAbandoned, reason)
	})
}

// transitionTask parses the task ID, opens the epic, and performs the status transition.
func transitionTask(rawID, epicsDir string, to epic.Status, reason string) error {
	taskID, err := epic.ParseTaskID(rawID)
	if err != nil {
		fmt.Println(epic.JSONError(err))
		return err
	}
	conn, q, err := epic.OpenEpic(taskID.Root(), epicsDir)
	if err != nil {
		fmt.Println(epic.JSONError(err))
		return err
	}
	defer conn.Close()
	ctx := context.Background()
	if err := epic.TransitionStatus(ctx, conn, q, taskID, to, reason); err != nil {
		fmt.Println(epic.JSONError(err))
		return err
	}
	fmt.Println(epic.JSONSuccess(nil))
	return nil
}
