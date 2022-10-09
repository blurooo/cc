// Package plugin 提供动态挂载插件的核心能力
// 实现 tc 命令引擎的执行能力
// 目前命令的实现方式包含了
// 1. github actions 流水线即命令，通过 ga 实现
// 2. tcli 插件即命令
package plugin_bak

import (
	"context"
	"errors"

	"github.com/blurooo/cc/cli"
	"github.com/blurooo/cc/plugin/schemas"
)

// ErrUnSupported 不支持的错误类型
var ErrUnSupported = errors.New("不支持的类型")

// Info 插件信息
type Info struct {
	Name    string
	Desc    string
	Version string
}

// UpdateOpts 插件更新相关信息
type UpdateOpts struct {
	// Lazy 懒更新模式，只是标记更新，而不会立即更新
	Lazy bool
}

// LoadOpts 加载参数
type LoadOpts struct {
	// Update 是否执行更新
	Update bool
}

// Plugin 插件抽象实现，导出，可被外部感知的
type Plugin interface {
	// Load 加载插件
	Load(ctx context.Context, opts LoadOpts) error
	// Update 更新插件
	Update(opts UpdateOpts) error
	// Exec 运行插件
	Exec(ctx context.Context, opts schemas.ExecOpts) error
	// Info 获取插件信息
	Info() Info
	// Scenes 获取插件场景
	Scenes() []Scene
}

type Command struct {
	Path              string
	CommandSourcePath string
	PluginRootPath    string
	Plugin            schemas.Plugin
}

func (c *Command) Load(ctx context.Context, opts LoadOpts) error {
	return loadPlugin(ctx, c.Plugin)
}

func loadPlugin(ctx context.Context, plugin schemas.Plugin) error {
	depPlugins, err := plugin.Plugins()
	if err != nil {
		return err
	}
	for _, p := range depPlugins {
		if err := loadPlugin(ctx, p); err != nil {
			return err
		}
	}
	if err := plugin.Resource().Load(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Command) Update(opts UpdateOpts) error {
	// TODO implement me
	panic("implement me")
}

func (c *Command) Exec(ctx context.Context, opts schemas.ExecOpts) error {
	return c.Plugin.Enter().Exec(ctx, opts)
}

func (c *Command) Info() Info {
	return Info{
		Name:    c.Plugin.Name(),
		Desc:    c.Plugin.Desc(),
		Version: c.Plugin.Version(),
	}
}

func (c *Command) Scenes() []Scene {
	// TODO implement me
	panic("implement me")
}

func (c *Command) getPlugin() (schemas.Plugin, error) {
	enter := schemas.singlePlugin{
		RootPath: c.PluginRootPath,
	}
	return enter.ResolvePath(c.Path)
}

type Flow struct{}

// NewPlugin 创建插件实例
func NewPlugin(cli cli.Executor, path string) (Plugin, error) {
	tCLIPlugin, err := newTCLIPlugin(cli, path)
	if err == nil {
		return tCLIPlugin, err
	}
	if err != ErrUnSupported {
		return nil, err
	}
	return newGaPlugin(cli, path)
}
