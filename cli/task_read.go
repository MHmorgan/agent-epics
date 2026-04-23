package cli

import (
	"context"
	"fmt"

	clilib "github.com/Minimal-Viable-Software/cli-go"

	"github.com/MHmorgan/agent-epics/epic"
)

// RegisterTaskReadCommands registers task:get, task:list, task:context:get,
// task:records, task:next.
func RegisterTaskReadCommands(app *clilib.Application, epicsDir string) {
	registerTaskGetCmd(app, epicsDir)
	registerTaskListCmd(app, epicsDir)
	registerTaskContextGetCmd(app, epicsDir)
	registerTaskRecordsCmd(app, epicsDir)
	registerTaskNextCmd(app, epicsDir)
}

func registerTaskGetCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	cmd := app.SubCommand("task:get", "Get a task")
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

		task, err := epic.GetTask(context.Background(), q, taskID)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(task))
		return nil
	})
}

func registerTaskListCmd(app *clilib.Application, epicsDir string) {
	var epicArg string
	var showAll bool
	var parentFlag string

	cmd := app.SubCommand("task:list", "List tasks in an epic")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.BoolFlag(&showAll, "all", "Include terminal tasks")
	cmd.StringFlag(&parentFlag, "parent", "Filter to children of this ID")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, epicsDir)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		defer conn.Close()

		var parentID epic.TaskID
		if parentFlag != "" {
			parentID, err = epic.ParseTaskID(parentFlag)
			if err != nil {
				fmt.Println(epic.JSONError(err))
				return err
			}
		}

		tasks, err := epic.ListTasks(context.Background(), q, parentID, showAll)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(tasks))
		return nil
	})
}

func registerTaskContextGetCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	cmd := app.SubCommand("task:context:get", "Get composed context")
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

		composed, err := epic.ComposeContext(context.Background(), q, taskID)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(map[string]string{"context": composed}))
		return nil
	})
}

func registerTaskRecordsCmd(app *clilib.Application, epicsDir string) {
	var rawID string
	var selfOnly bool

	cmd := app.SubCommand("task:records", "Get records")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.BoolFlag(&selfOnly, "self", "Exact match only instead of subtree")
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

		records, err := epic.GetRecords(context.Background(), q, taskID, selfOnly)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(records))
		return nil
	})
}

func registerTaskNextCmd(app *clilib.Application, epicsDir string) {
	var epicArg string
	cmd := app.SubCommand("task:next", "Get next ready task")
	cmd.StringArg(&epicArg, "epic", "Epic ID")
	cmd.Run(func() error {
		conn, q, err := epic.OpenEpic(epicArg, epicsDir)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		defer conn.Close()

		task, err := epic.NextTask(context.Background(), conn, q, epicArg)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(task))
		return nil
	})
}
