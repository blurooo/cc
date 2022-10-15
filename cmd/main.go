package main

import (
	mixer "github.com/blurooo/cc"
	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/config"
)

func main() {
	app := config.Application{
		Name:      "mixer",
		Desc:      "Devops 工具",
		Debug:     false,
		Version:   "v1.0.0",
		GroupName: "mixer",
		Flags: config.Flags{
			EnableConfig:  true,
			EnableInstall: true,
			EnableDaemon:  true,
			EnableDynamic: true,
		},
		InitPersistentConfig: config.PersistentConfig{
			Command: config.Command{Repo: "https://github.com/modern-devops/devops-plugins.git"},
		},
	}
	m, err := mixer.NewMixedCommandLineTool(app)
	if err != nil {
		panic(err)
	}
	source := command.Source{
		Workspace:    m.WorkspaceRootPath,
		App:          m.App,
		Configurator: m.Configurator,
	}
	err = m.Start([]command.SourceLoader{source.ConfigSource})
	if err != nil {
		panic(err)
	}
}
