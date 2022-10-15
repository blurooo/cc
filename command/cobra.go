package command

import (
	"context"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/log"
	"github.com/blurooo/cc/plugin"
)

type CobraCommands struct {
	App           config.Application
	SourceLoaders []SourceLoader
}

func (c *CobraCommands) Nodes() ([]Node, error) {
	var searchers []Searcher
	for _, sl := range c.SourceLoaders {
		ss, err := sl()
		if err != nil {
			return nil, err
		}
		searchers = append(searchers, ss...)
	}
	if len(searchers) == 0 {
		return nil, nil
	}
	var nodes []Node
	var filter map[string]bool
	for _, searcher := range searchers {
		ns, err := searcher.List()
		if err != nil {
			return nil, err
		}
		for _, n := range ns {
			if _, ok := filter[n.FullName()]; ok {
				continue
			}
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (c *CobraCommands) ExecFile(path string, args []string) error {
	// 属于依赖调用时，应屏蔽依赖的所有不相关的标准输出
	// TODO(blurooochen): 保留文件输出
	c.App.Logger = log.Discard
	resolver := &plugin.Resolver{
		Name:           c.App.Name,
		PluginRootPath: c.App.WorkspaceLayout.PluginRootPath,
	}
	p, err := resolver.ResolvePath(context.Background(), path)
	if err != nil {
		return err
	}
	return c.ExecPlugin(p, args)
}

func (c *CobraCommands) ExecNode(node Node, args []string) error {
	return c.ExecPlugin(node.Plugin, args)
}

func (c *CobraCommands) ExecPlugin(p plugin.Plugin, args []string) error {
	if err := p.Load(context.Background(), plugin.LoadOpts{Update: false, Lazy: true}); err != nil {
		return err
	}
	return p.Execute(context.Background(), plugin.ExecOpts{
		Args: args,
	})
}
