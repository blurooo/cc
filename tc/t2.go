// Package tc 核心业务逻辑组装
package tc

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/pkg/linker"
	"github.com/blurooo/cc/plugin"
)

var cliClient = cli.Local()

// Nodes 获取指令节点
func Nodes() ([]command.Node, error) {
	searcher := getSearcher(LoadConfig().Update.Always, config.CommandDir)
	return searcher.List()
}

// ExecFile 执行指定插件文件
func ExecFile(pluginPath string, args []string) error {
	// 属于依赖调用时，应屏蔽依赖的所有不相关的标准输出
	// TODO(blurooochen): 保留文件输出
	ioc.Log.SetOutput(ioutil.Discard)
	entry := reporter.NewEntry(args)
	entry.Command = pluginPath
	var err error
	var p plugin.Plugin
	defer func() {
		addEntry(entry, err)
	}()
	p, err = plugin.NewPlugin(cliClient, pluginPath)
	if err != nil {
		return err
	}
	info := p.Info()
	entry.Command = info.Name
	entry.Version = info.Version
	// 插件不执行更新
	err = p.Load(nil, plugin.LoadOpts{Update: false})
	if err != nil {
		return err
	}
	err = p.Exec(nil, plugin.ExecOpts{
		Args: args,
	})
	return err
}

// ExecNode 执行指定节点
func ExecNode(node command.Node, args []string) error {
	err := node.Plugin.Load(nil, plugin.LoadOpts{Update: LoadConfig().Update.Always})
	if err != nil {
		return err
	}
	return node.Plugin.Exec(nil, plugin.ExecOpts{
		Args: args,
	})
}

// InstallFile 安装文件为命令
func InstallFile(pluginPath string) error {
	p, err := plugin.NewPlugin(cliClient, pluginPath)
	if err != nil {
		return fmt.Errorf("插件解析失败：%w", err)
	}
	err = p.Load(nil, plugin.LoadOpts{Update: true})
	if err != nil {
		return fmt.Errorf("资源加载失败：%w", err)
	}
	binPath := filepath.Join(config.AppConfDir, config.BinPath)
	execCommand := fmt.Sprintf(`%s exec "%s"`, config.AliasName, pluginPath)
	linkPath, err := linker.New(p.Info().Name, binPath, execCommand, linker.OverrideAlways)
	if err != nil {
		return err
	}
	ioc.Log.Infof("已成功连接到 %s，请确保 %s 已被添加到环境变量 PATH 中", linkPath, binPath)
	return nil
}

// InstallableList 获取可安装列表
func InstallableList() ([]command.Node, error) {
	searcher := getSearcher(LoadConfig().Update.Always, config.InstallDir)
	return searcher.List()
}

func getSearcher(autoUpdate bool, commandDir string) command.Searcher {
	c := LoadConfig()
	if c.Command.Path != "" {
		return command.FileSearcher(c.Command.Path, commandDir)
	}
	repoURL := config.CommandRepoURL
	if c.Command.Repo != "" {
		repoURL = c.Command.Repo
	}
	repoEngine := &repo.Repo{RepoStashDir: config.RepoStashDir, AutoUpdate: autoUpdate}
	return command.RepoSearcher(repoURL, repoEngine, commandDir)
}

func addEntry(entry report.Entry, err error) {
	entry.Scene = config.SceneT2Dependence
	entry.End(err)
	ioc.Reporter.Add(entry)
}
