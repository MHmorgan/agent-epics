package cli

import (
	"context"
	"fmt"

	clilib "github.com/Minimal-Viable-Software/cli-go"

	"github.com/MHmorgan/agent-epics/epic"
)

// RegisterStructureCommands registers task:split, task:unsplit, task:after, task:unafter.
func RegisterStructureCommands(app *clilib.Application, epicsDir string) {
	registerSplitCmd(app, epicsDir)
	registerUnsplitCmd(app, epicsDir)
	registerAfterCmd(app, epicsDir)
	registerUnafterCmd(app, epicsDir)
}

func registerSplitCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	cmd := app.SubCommand("task:split", "Split a task into subtasks")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
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
		if err := epic.SplitTask(ctx, conn, q, taskID); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}

func registerUnsplitCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	cmd := app.SubCommand("task:unsplit", "Unsplit a task")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
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
		if err := epic.UnsplitTask(ctx, conn, q, taskID); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}

func registerAfterCmd(app *clilib.Application, epicsDir string) {
	var rawID, rawPred string
	cmd := app.SubCommand("task:after", "Add a dependency (task after predecessor)")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&rawPred, "pred", "Predecessor task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		predID, err := epic.ParseTaskID(rawPred)
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
		if err := epic.AddDependency(ctx, q, taskID, predID); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}

func registerUnafterCmd(app *clilib.Application, epicsDir string) {
	var rawID, rawPred string
	cmd := app.SubCommand("task:unafter", "Remove a dependency")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&rawPred, "pred", "Predecessor task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		predID, err := epic.ParseTaskID(rawPred)
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
		if err := epic.RemoveDependency(ctx, q, taskID, predID); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}
