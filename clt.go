package blurooo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/flags"
	"github.com/blurooo/cc/log"
)

type MixedCommandLineTool struct {
	App          config.Application
	Configurator *config.Configurator
}

// NewMixedCommandLineTool create a new mixed command line tool.
func NewMixedCommandLineTool(app config.Application) (*MixedCommandLineTool, error) {
	if app.WorkspaceLayout.RootPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		app.WorkspaceLayout.RootPath = filepath.Join(home, "."+app.Name)
	}
	if app.CommandDirectory == "" {
		app.CommandDirectory = "cmd"
	}
	if err := os.MkdirAll(app.WorkspaceLayout.RootPath, os.ModeDir); err != nil {
		return nil, fmt.Errorf("create workspace [%s] failed, %w", app.WorkspaceLayout.RootPath, err)
	}
	if os.Getenv("DEBUG") == "true" {
		app.Debug = true
	}
	app.Logger = log.New(app.Debug)
	app.WorkspaceLayout = config.BuildWorkspaceLayout(app.WorkspaceLayout)
	configurator, err := config.NewConfigurator(app.WorkspaceLayout.ConfigFile, app.InitPersistentConfig)
	if err != nil {
		return nil, err
	}
	return &MixedCommandLineTool{App: app, Configurator: configurator}, nil
}

func (m *MixedCommandLineTool) Exec() error {
	f := flags.Flags{Config: m.App}
	return f.Execute()
}
