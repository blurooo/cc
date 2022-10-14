package command

import (
	"github.com/blurooo/cc/config"
)

type CobraCommands struct {
	Config       config.Application
	Configurator *config.Configurator
}

func (c *CobraCommands) Nodes() ([]Node, error) {
	searcher := getSearcher(c.Configurator.LoadConfig().Update.Always, config.CommandDir)
	return searcher.List()
}
