package blurooo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/blurooo/cc/config"
)

type MixedCommandLineTool struct {
	App          config.ApplicationConfig
	Configurator *config.Configurator
}

// NewMixedCommandLineTool create a new mixed command line tool.
func NewMixedCommandLineTool(app config.ApplicationConfig) (*MixedCommandLineTool, error) {
	if app.WorkspaceLayout.RootPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		app.WorkspaceLayout.RootPath = filepath.Join(home, "."+app.Name)
	}
	if err := os.MkdirAll(app.WorkspaceLayout.RootPath, os.ModeDir); err != nil {
		return nil, fmt.Errorf("create workspace [%s] failed, %w", app.WorkspaceLayout.RootPath, err)
	}
	if os.Getenv("DEBUG") == "true" {
		app.Debug = true
	}
	app.WorkspaceLayout = config.BuildWorkspaceLayout(app.WorkspaceLayout)
	configurator, err := config.NewConfigurator(app.WorkspaceLayout.ConfigFile, app.DefaultPersistentConfig)
	if err != nil {
		return nil, err
	}
	return &MixedCommandLineTool{App: app, Configurator: configurator}, nil
}
