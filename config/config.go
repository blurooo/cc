package config

import (
	"path/filepath"

	"github.com/blurooo/cc/log"
	"github.com/spf13/cobra"
)

type ApplicationConfig struct {
	Name                    string
	Desc                    string
	Debug                   bool
	Version                 string
	Logger                  log.Logger
	Flags                   Flags
	WorkspaceLayout         WorkspaceLayout
	DefaultPersistentConfig PersistentConfig
}

type Hook struct {
	OnInitialize         func()
	OnPreRegisterCommand func(command *cobra.Command) error
}

type Flags struct {
	EnableConfig  bool
	EnableInstall bool
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

func BuildWorkspaceLayout(layout WorkspaceLayout) WorkspaceLayout {
	layout.PluginRootPath = layout.getOrDefault(layout.PluginRootPath, "plugin")
	layout.ResourceRootPath = layout.getOrDefault(layout.ResourceRootPath, "resource")
	layout.RepoRootPath = layout.getOrDefault(layout.RepoRootPath, "repo")
	layout.DaemonPath = layout.getOrDefault(layout.DaemonPath, "daemon")
	layout.LogPath = layout.getOrDefault(layout.LogPath, "log")
	layout.BinPath = layout.getOrDefault(layout.BinPath, "bin")
	layout.ConfigFile = layout.getOrDefault(layout.ConfigFile, "config")
	return layout
}

func (l *WorkspaceLayout) getOrDefault(path, defaultPath string) string {
	if path == "" {
		return filepath.Join(l.RootPath, defaultPath)
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.RootPath, path)
}
