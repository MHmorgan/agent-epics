package cli

import (
	"context"
	"fmt"

	clilib "github.com/Minimal-Viable-Software/cli-go"

	"github.com/MHmorgan/agent-epics/epic"
)

// RegisterTaskWriteCommands registers task:new-epic, task:add-child, task:set,
// task:context:set, task:record.
func RegisterTaskWriteCommands(app *clilib.Application, epicsDir string) {
	registerNewEpicCmd(app, epicsDir)
	registerAddChildCmd(app, epicsDir)
	registerSetCmd(app, epicsDir)
	registerContextSetCmd(app, epicsDir)
	registerRecordCmd(app, epicsDir)
}

func registerNewEpicCmd(app *clilib.Application, epicsDir string) {
	var epicID string
	cmd := app.SubCommand("task:new-epic", "Create a new epic")
	cmd.StringArg(&epicID, "epic", "Epic ID")
	cmd.Run(func() error {
		if err := epic.NewEpic(epicID, epicsDir); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(map[string]string{"id": epicID}))
		return nil
	})
}

func registerAddChildCmd(app *clilib.Application, epicsDir string) {
	var parent string
	cmd := app.SubCommand("task:add-child", "Add child to branch")
	cmd.StringArg(&parent, "parent", "Parent task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(parent)
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

		childID, err := epic.AddChild(context.Background(), conn, q, taskID)
		if err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(map[string]string{"id": childID.String()}))
		return nil
	})
}

func registerSetCmd(app *clilib.Application, epicsDir string) {
	var rawID, markdown string
	cmd := app.SubCommand("task:set", "Set task body")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&markdown, "markdown", "Body text")
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

		if err := epic.SetTaskBody(context.Background(), conn, q, taskID, markdown); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}

func registerContextSetCmd(app *clilib.Application, epicsDir string) {
	var rawID, markdown string
	cmd := app.SubCommand("task:context:set", "Set task context")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&markdown, "markdown", "Context text")
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

		if err := epic.SetTaskContext(context.Background(), conn, q, taskID, markdown); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}

func registerRecordCmd(app *clilib.Application, epicsDir string) {
	var rawID, text string
	cmd := app.SubCommand("task:record", "Append agent record")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.StringArg(&text, "text", "Record text")
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

		if err := epic.AddRecord(context.Background(), q, taskID, text); err != nil {
			fmt.Println(epic.JSONError(err))
			return err
		}
		fmt.Println(epic.JSONSuccess(nil))
		return nil
	})
}
