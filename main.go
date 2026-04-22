package main

import (
	"context"
	"os"

	"github.com/MHmorgan/agent-epics/common"
	"github.com/MHmorgan/agent-epics/db"
	"github.com/Minimal-Viable-Software/log-go"
)

func main() {
	ctx := context.Background()

	// Setup config
	cfg := common.LoadConfig()
	ctx = context.WithValue(ctx, "config", &cfg)

	// Ensure appdir subdirectories exist
	if err := os.MkdirAll(cfg.EpicsDir, 0o755); err != nil {
		log.Fatalln("create repos dir:", err)
	}

	// @Todo run CLI
}
