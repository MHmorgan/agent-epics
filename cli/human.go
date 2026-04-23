package cli

import (
	"fmt"
	"os"

	clilib "github.com/Minimal-Viable-Software/cli-go"

	"github.com/MHmorgan/agent-epics/epic"
)

// RegisterHumanCommands registers epics, rm, purge commands on the app.
func RegisterHumanCommands(app *clilib.Application, epicsDir string) {
	registerEpicsCmd(app, epicsDir)
	registerRmCmd(app, epicsDir)
	registerPurgeCmd(app, epicsDir)
}

func registerEpicsCmd(app *clilib.Application, epicsDir string) {
	cmd := app.SubCommand("epics", "List all epics")
	cmd.Run(func() error {
		infos, err := epic.ListEpics(epicsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		if len(infos) == 0 {
			fmt.Println("No epics found.")
			return nil
		}
		for _, info := range infos {
			fmt.Printf("%s  %s\n", info.ID, info.Status)
		}
		return nil
	})
}

func registerRmCmd(app *clilib.Application, epicsDir string) {
	var epicID string
	cmd := app.SubCommand("rm", "Remove an epic")
	cmd.StringArg(&epicID, "epic", "Epic ID to remove")
	cmd.Run(func() error {
		if err := epic.RemoveEpic(epicID, epicsDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		fmt.Printf("Removed %s\n", epicID)
		return nil
	})
}

func registerPurgeCmd(app *clilib.Application, epicsDir string) {
	cmd := app.SubCommand("purge", "Remove all terminal epics")
	cmd.Run(func() error {
		purged, err := epic.PurgeTerminalEpics(epicsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		if len(purged) == 0 {
			fmt.Println("No terminal epics to purge.")
			return nil
		}
		for _, id := range purged {
			fmt.Printf("Purged %s\n", id)
		}
		return nil
	})
}
