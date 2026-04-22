package common

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/Minimal-Viable-Software/config-go"
	"github.com/Minimal-Viable-Software/log-go"
)

const ConfigPrefix = "AE_"

func init() {
	config.SetPrefix(ConfigPrefix)
	config.Value(&log.Lvl, "LOGLEVEL")
}

// Config holds all application configuration derived from environment variables.
type Config struct {
	AppDir   string // Agent epics work directory
	EpicsDir string // Directory with all epic files
}

// LoadConfig reads environment variables and returns a Config.
func LoadConfig() Config {
	cfg := Config{
		AppDir: os.ExpandEnv("$HOME/.agent-epics"),
	}

	config.String(&cfg.AppDir, "APPDIR")

	cfg.EpicsDir = filepath.Join(cfg.AppDir, "epics")

	return cfg
}

// GetConfig returns the [Config] object from the "config" context value, or panics.
func GetConfig(ctx context.Context) *Config {
	value := ctx.Value("config")
	if value == nil {
		panic("config not found in context")
	}
	db, ok := value.(*Config)
	if !ok {
		t := reflect.TypeOf(value)
		s := fmt.Sprintf("failed to cast db (of type %v) to *Config", t)
		panic(s)
	}
	return db
}
