package cli

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/common"
	"github.com/MHmorgan/agent-epics/epic"
	"github.com/Minimal-Viable-Software/log-go"
)

// registerHumanCommands registers epics, rm, purge, show, context commands on the app.
func registerHumanCommands(ctx context.Context) {
	epicsCmd(ctx)
	rmCmd(ctx)
	purgeCmd(ctx)
	showCmd(ctx)
	contextCmd(ctx)
}

func epicsCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)

	cmd := app.SubCommand("epics", "List all epics")

	cmd.Run(func() error {
		infos, err := epic.ListEpics(cfg.EpicsDir)
		if err != nil {
			return fmt.Errorf("list epics: %w", err)
		}
		if len(infos) == 0 {
			log.Infoln("No epics found.")
			return nil
		}
		for _, info := range infos {
			fmt.Printf("%s  %s\n", info.ID, info.Status)
		}
		return nil
	})
}

func rmCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	cmd := app.SubCommand("rm", "Remove an epic")

	var epicID string
	cmd.StringArg(&epicID, "epic", "Epic ID to remove")

	cmd.Run(func() error {
		if err := epic.RemoveEpic(epicID, cfg.EpicsDir); err != nil {
			return fmt.Errorf("remove epic: %w", err)
		}
		log.Infoln("Removed", epicID)
		return nil
	})
}

func purgeCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	cmd := app.SubCommand("purge", "Remove all terminal epics")

	cmd.Run(func() error {
		purged, err := epic.PurgeTerminalEpics(cfg.EpicsDir)
		if err != nil {
			return fmt.Errorf("purge epics: %w", err)
		}
		if len(purged) == 0 {
			log.Infoln("No terminal epics to purge.")
			return nil
		}
		for _, id := range purged {
			log.Infoln("Purged", id)
		}
		return nil
	})
}

func showCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID string
	cmd := app.SubCommand("show", "Print task body")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			return fmt.Errorf("show: %w", err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return fmt.Errorf("show: %w", err)
		}
		defer conn.Close()

		task, err := epic.GetTask(ctx, q, taskID)
		if err != nil {
			return fmt.Errorf("show: %w", err)
		}
		fmt.Print(task.Body)
		return nil
	})
}

func contextCmd(ctx context.Context) {
	cfg := common.GetConfig(ctx)
	var rawID string
	cmd := app.SubCommand("context", "Print composed context")
	cmd.StringArg(&rawID, "id", "Task ID")
	cmd.Run(func() error {
		taskID, err := epic.ParseTaskID(rawID)
		if err != nil {
			return fmt.Errorf("context: %w", err)
		}
		conn, q, err := epic.OpenEpic(taskID.Root(), cfg.EpicsDir)
		if err != nil {
			return fmt.Errorf("context: %w", err)
		}
		defer conn.Close()

		composed, err := epic.ComposeContext(ctx, q, taskID)
		if err != nil {
			return fmt.Errorf("context: %w", err)
		}
		fmt.Print(composed)
		return nil
	})
}
