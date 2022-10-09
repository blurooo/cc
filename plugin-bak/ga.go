package plugin_bak

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"tencent2/tools/dev_tools/t2cli/utils/cli"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/ioc"
	"github.com/blurooo/cc/util/cslice"
)

const gaCommand = "ga"
const inputParamsPrefix = "INPUT_PARAMS_"

// githubActionsPlugin 基于 github actions 的本地流水线引擎
type githubActionsPlugin struct {
	Workflow Workflow
	Path     string
	Cli      cli.Executor
	Plugin   *tCLIPlugin
}

// newGaPlugin 从文件得到插件实例
func newGaPlugin(cli cli.Executor, path string) (*githubActionsPlugin, error) {
	if !isGaFile(path) {
		return nil, ErrUnSupported
	}
	workflow, err := ResolveWorkflow(path)
	if err != nil {
		return nil, err
	}
	// 目前只用特征识别
	if len(workflow.Jobs) == 0 {
		return nil, ErrUnSupported
	}
	plugin, err := newTCLIPlugin(cli, path, skipCheckPluginFeature)
	if err != nil {
		return nil, err
	}
	return &githubActionsPlugin{
		Plugin:   plugin,
		Path:     path,
		Cli:      cli,
		Workflow: *workflow,
	}, nil
}

// Load 加载流水线插件
func (p *githubActionsPlugin) Load(ctx context.Context, opts LoadOpts) error {
	return p.Plugin.Load(nil, params)
}

// Update 更新插件
func (p *githubActionsPlugin) Update(info UpdateOpts) error {
	return p.Plugin.Update(info)
}

// Info 获取插件基本信息
func (p *githubActionsPlugin) Info() Info {
	return p.Plugin.Info()
}

// Scenes 获取 ga 插件场景
func (p *githubActionsPlugin) Scenes() []Scene {
	return p.Workflow.Scenes
}

// Exec 通过 GA 引擎执行流水线
func (p *githubActionsPlugin) Exec(ctx context.Context, opts ExecOpts) error {
	args := []string{gaCommand}
	if !config.Flags.Inquire {
		args = append(args, "-i", "false")
	}
	// 环境变量传递
	for _, env := range params.Args {
		k, v := toKV(env)
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, p.Path)
	shell := cli.QuoteCommand(args)
	ioc.Log.Infof("实际执行语句：%s", shell)
	// 使用 shell 的方式运行，可以使传递的 env 起作用
	return p.Cli.RunParamsInherit(context.TODO(), cli.Params{
		Shell: shell,
		Env:   p.Plugin.Envs,
	})
}

func toKV(env string) (string, string) {
	strList := strings.Split(env, "=")
	k := strList[0]
	v := strings.Join(strList[1:], "=")
	return inputParamsPrefix + strings.ToUpper(k), v
}

// isGaFile 是否GA指令文件
func isGaFile(file string) bool {
	ext := filepath.Ext(file)
	return cslice.IncludeString(fileExtList, ext)
}
