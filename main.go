package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/MHmorgan/agent-epics/cli"
	"github.com/MHmorgan/agent-epics/common"
	"github.com/Minimal-Viable-Software/log-go"
)

func main() {
	ctx := context.Background()

	// Setup config
	cfg := common.LoadConfig()
	ctx = context.WithValue(ctx, "config", &cfg)

	// Ensure appdir subdirectories exist
	if err := os.MkdirAll(cfg.EpicsDir, 0o755); err != nil {
		log.Fatalln("create epics dir:", err)
	}

	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		var ae cli.AgentError
		if errors.As(err, &ae) {
			fmt.Println(ae)
			os.Exit(1)
		}
		log.Fatalln(err)
	}
}
