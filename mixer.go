package mixer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/flags"
	"github.com/blurooo/cc/log"
	"github.com/blurooo/cc/tools/git"
)

type Mixer struct {
	App               config.Application
	Configurator      *config.Configurator
	WorkspaceRootPath string
}

// NewMixedCommandLineTool create a new mixed command line tool.
func NewMixedCommandLineTool(app config.Application) (*Mixer, error) {
	g, err := git.Instance("")
	if err != nil {
		return nil, err
	}
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
	if app.Logger == nil {
		app.Logger = log.New(app.Debug)
	}
	app.WorkspaceLayout = config.BuildWorkspaceLayout(app.WorkspaceLayout)
	configurator, err := config.NewConfigurator(app.WorkspaceLayout.ConfigFile, app.InitPersistentConfig)
	if err != nil {
		return nil, err
	}
	return &Mixer{App: app, Configurator: configurator, WorkspaceRootPath: g.RootPath()}, nil
}

func (m *Mixer) Start(sourceLoaders []command.SourceLoader) error {
	f := flags.Flags{App: m.App, CobraCommands: command.CobraCommands{App: m.App, SourceLoaders: sourceLoaders}}
	return f.Execute()
}
