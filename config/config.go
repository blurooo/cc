package config

import (
	"path/filepath"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/log"
	"github.com/spf13/cobra"
)

const (
	EnvSource = "CC_SOURCE_REPO"
)

type Application struct {
	Name                 string
	Desc                 string
	Debug                bool
	Version              string
	CommandDirectory     string
	GroupName            string
	SourceLoaders        []command.SourceLoader
	Logger               log.Logger
	Flags                Flags
	Handler              Handler
	WorkspaceLayout      WorkspaceLayout
	InitPersistentConfig PersistentConfig
}

type Handler struct {
	OnInitialize         func()
	OnPreRegisterCommand func(command *cobra.Command) error
}

type Flags struct {
	EnableConfig  bool
	EnableInstall bool
	EnableDaemon  bool
	EnableDynamic bool
}

type WorkspaceLayout struct {
	RootPath         string // default ~/.{name}
	BinPath          string
	LogPath          string
	DaemonPath       string
	RepoRootPath     string
	PluginRootPath   string
	ResourceRootPath string
	ConfigFile       string
}

const (
	dirNamePlugin   = "plugin"
	dirNameResource = "resource"
	dirNameRepo     = "repo"
	dirNameDaemon   = "daemon"
	dirNameLog      = "log"
	dirNameBin      = "bin"
	fileNameConfig  = "config"
)

func BuildWorkspaceLayout(layout WorkspaceLayout) WorkspaceLayout {
	layout.PluginRootPath = layout.getOrDefaultPath(layout.PluginRootPath, dirNamePlugin)
	layout.ResourceRootPath = layout.getOrDefaultPath(layout.ResourceRootPath, dirNameResource)
	layout.RepoRootPath = layout.getOrDefaultPath(layout.RepoRootPath, dirNameRepo)
	layout.DaemonPath = layout.getOrDefaultPath(layout.DaemonPath, dirNameDaemon)
	layout.LogPath = layout.getOrDefaultPath(layout.LogPath, dirNameLog)
	layout.BinPath = layout.getOrDefaultPath(layout.BinPath, dirNameBin)
	layout.ConfigFile = layout.getOrDefaultPath(layout.ConfigFile, fileNameConfig)
	return layout
}

func (l *WorkspaceLayout) getOrDefaultPath(path, defaultPath string) string {
	if path == "" {
		return filepath.Join(l.RootPath, defaultPath)
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.RootPath, path)
}
