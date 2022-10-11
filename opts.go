package blurooo

import (
	"fmt"
	"os"
	"path/filepath"
)

type Opts struct {
	Name    string
	Debug   bool
	Version string
	Layout  Layout
}

type Layout struct {
	RootPath         string // default ~/.{name}
	BinPath          string
	LogPath          string
	DaemonPath       string
	RepoRootPath     string
	PluginRootPath   string
	ResourceRootPath string
	ConfigFile       string
}

type CC struct {
	Opts
}

func New(opts Opts) (*CC, error) {
	if opts.Layout.RootPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		opts.Layout.RootPath = filepath.Join(home, "."+opts.Name)
	}
	if err := os.MkdirAll(opts.Layout.RootPath, os.ModeDir); err != nil {
		return nil, fmt.Errorf("create workspace [%s] failed, %w", opts.Layout.RootPath, err)
	}
	opts.Layout = buildLayout(opts.Layout)
	return &CC{opts}, nil
}

func buildLayout(layout Layout) Layout {
	layout.PluginRootPath = layout.getOrDefault(layout.PluginRootPath, "plugin")
	layout.ResourceRootPath = layout.getOrDefault(layout.ResourceRootPath, "resource")
	layout.RepoRootPath = layout.getOrDefault(layout.RepoRootPath, "repo")
	layout.DaemonPath = layout.getOrDefault(layout.DaemonPath, "daemon")
	layout.LogPath = layout.getOrDefault(layout.LogPath, "log")
	layout.BinPath = layout.getOrDefault(layout.BinPath, "bin")
	layout.ConfigFile = layout.getOrDefault(layout.ConfigFile, "config")
	return layout
}

func (l *Layout) getOrDefault(path, defaultPath string) string {
	if path == "" {
		return filepath.Join(l.RootPath, defaultPath)
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.RootPath, path)
}
