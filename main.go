package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/MHmorgan/agent-epics/cli"
	"github.com/MHmorgan/agent-epics/common"
	clilib "github.com/Minimal-Viable-Software/cli-go"
	"github.com/Minimal-Viable-Software/log-go"
)

func main() {
	// Setup config
	cfg := common.LoadConfig()

	// Ensure appdir subdirectories exist
	if err := os.MkdirAll(cfg.EpicsDir, 0o755); err != nil {
		log.Fatalln("create epics dir:", err)
	}

	// Setup CLI
	app := clilib.NewApplication()

	cli.RegisterHumanCommands(app, cfg.EpicsDir)
	cli.RegisterTaskWriteCommands(app, cfg.EpicsDir)
	cli.RegisterTaskReadCommands(app, cfg.EpicsDir)
	cli.RegisterStructureCommands(app, cfg.EpicsDir)
	cli.RegisterStatusCommands(app, cfg.EpicsDir)
	cli.RegisterAttrCommands(app, cfg.EpicsDir)

	// Run
	if err := app.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, clilib.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
