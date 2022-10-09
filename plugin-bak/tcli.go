package plugin_bak

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
	"tencent2/tools/dev_tools/t2cli/utils/cli"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/errs"
	"github.com/blurooo/cc/ioc"
	"github.com/blurooo/cc/pkg/linker"
	"github.com/blurooo/cc/resource"
	"github.com/blurooo/cc/util/cruntime"
	"github.com/blurooo/cc/util/cslice"
	"github.com/blurooo/cc/util/git"
	"github.com/blurooo/cc/util/option"
	"github.com/blurooo/cc/util/path"
)

const (
	// None 什么操作都没有
	_ option.Option = iota
	// skipCheckPluginFeature 跳过检查插件特征
	skipCheckPluginFeature = 1 << 0
)

// tCLIPlugin 基于 tcli 协议实现
type tCLIPlugin struct {
	// TCLISchema 插件信息
	Schema TCLISchema

	// 插件工作目录
	WorkDir string
	// 协议完整路径
	Path string
	// Cli 命令行执行器
	Cli cli.Executor
	// GitInstance 路径的GIT实例
	GitInstance *git.Git
	Envs        []string
}

// newTCLIPlugin 从协议文件获取执行器
func newTCLIPlugin(cli cli.Executor, path string, options ...option.Option) (*tCLIPlugin, error) {
	if !isTargetFile(path) {
		return nil, ErrUnSupported
	}
	schema, err := ResolveTCli(path)
	if err != nil {
		return nil, err
	}
	if !option.Has(options, skipCheckPluginFeature) {
		// 目前只用特征识别
		if len(schema.Command) == 0 {
			return nil, ErrUnSupported
		}
	}
	if len(schema.Dep.Os) > 0 && !cslice.IncludeString(schema.Dep.Os, cruntime.GOOS()) {
		return nil, ErrUnSupported
	}
	workDir := pluginWorkDir(schema.Name, schema.Version)
	gitInstance, err := git.Instance(path)
	if err != nil {
		return nil, fmt.Errorf("无法加载目录：%s，请确认是否受污染：%w", path, err)
	}
	return &tCLIPlugin{
		GitInstance: gitInstance,
		Path:        path,
		WorkDir:     workDir,
		Cli:         cli,
		Schema:      *schema,
	}, nil
}

// Update 更新插件
func (p *tCLIPlugin) Update(info UpdateOpts) error {
	willUpdate, err := p.needUpdate()
	if err != nil {
		return err
	}
	if !willUpdate {
		return nil
	}
	workDir := pluginWorkDir(p.Schema.Name, p.Schema.Version)
	doneFile := filepath.Join(workDir, config.DoneFile)
	if !info.Lazy {
		return p.loadPluginResource()
	}
	// 懒更新的情况下，确保 done 标志不存在即可
	if !cfile.Exist(doneFile) {
		return nil
	}
	err = os.Remove(doneFile)
	if err != nil {
		return fmt.Errorf("移除旧版本失败：%s", err)
	}
	return nil
}

// Load 加载插件
func (p *tCLIPlugin) Load(ctx context.Context, opts LoadOpts) error {
	return p.load(params.Update)
}

// Exec 执行插件
func (p *tCLIPlugin) Exec(ctx context.Context, opts ExecOpts) error {
	commands := p.Schema.Commands(p.WorkDir)
	return runCommands(commands, p.Envs, params.Args)
}

// Info 获取插件信息
func (p *tCLIPlugin) Info() Info {
	return Info{
		Name:    p.Schema.Name,
		Desc:    p.Schema.Desc,
		Version: p.Schema.Version,
	}
}

// Scenes 获取 tcli 插件场景
func (p *tCLIPlugin) Scenes() []Scene {
	return nil
}

// load 加载插件，插件已存在时跳过
func (p *tCLIPlugin) load(update bool) error {
	err := p.prepareEnvs()
	if err != nil {
		return err
	}
	doneFile := filepath.Join(p.WorkDir, config.DoneFile)
	if !cfile.Exist(doneFile) {
		return p.loadPluginResource()
	}
	if !update {
		return nil
	}
	willUpdate, err := p.needUpdate()
	if err != nil {
		return err
	}
	if !willUpdate {
		return nil
	}
	return p.loadPluginResource()
}

func (p *tCLIPlugin) needUpdate() (bool, error) {
	workDir := pluginWorkDir(p.Schema.Name, p.Schema.Version)
	doneFile := filepath.Join(workDir, config.DoneFile)
	if !cfile.Exist(doneFile) {
		return true, nil
	}
	data, err := os.ReadFile(doneFile)
	if err != nil {
		return false, err
	}
	doneInfo := &LoadDoneInfo{}
	err = json.Unmarshal(data, doneInfo)
	if err != nil {
		return false, err
	}
	schemaVersion, err := getSchemaVersion(p.GitInstance, p.Path)
	if err != nil {
		return false, err
	}
	// 协议版本或工具版本有变化都认为需要更新
	return doneInfo.Version != p.Schema.Version || doneInfo.SchemaVersion != schemaVersion, nil
}

func pluginWorkDir(name, version string) string {
	return filepath.Join(config.PluginDir, fmt.Sprintf("%s@%s", name, version))
}

// getSchemaVersion 获取文件协议版本
func getSchemaVersion(gitInstance *git.Git, path string) (string, error) {
	schemaVersion, err := gitInstance.LastChange(path)
	if err == nil {
		return schemaVersion, nil
	}
	if err == git.ErrNotFound {
		// 没有找到变更记录，说明文件一般为新建的，还没纳入版本管理，此时直接给出文件的md5码
		schemaVersion, err = cfile.MD5(path)
	}
	if err != nil {
		return "", err
	}
	return schemaVersion, nil
}

// isTargetFile 是否目标协议文件
func isTargetFile(file string) bool {
	ext := filepath.Ext(file)
	return cslice.IncludeString(fileExtList, ext)
}

// loadPluginResource 加载插件资源
func (p *tCLIPlugin) loadPluginResource() error {
	doneFile := filepath.Join(p.WorkDir, config.DoneFile)
	var err error
	defer func() {
		if err != nil {
			err = errs.NewProcessErrorWithCode(err, errs.CodePluginLoadFail)
		}
	}()
	err = os.RemoveAll(p.WorkDir)
	if err != nil {
		return fmt.Errorf("移除插件工作目录 [%s] 失败：%w", p.WorkDir, err)
	}
	err = cfile.MkdirAll(p.WorkDir)
	if err != nil {
		return fmt.Errorf("创建插件工作目录 [%s] 失败：%w", p.WorkDir, err)
	}
	err = p.handlePreLoad()
	if err != nil {
		return err
	}
	binPath := depBinPath(p.WorkDir)
	// 链式平铺式构建依赖
	for _, dep := range p.Schema.Dep.Plugin {
		err = p.buildDep(dep, p.GitInstance, binPath)
		if err != nil {
			return err
		}
	}
	r := &resource.Resource{
		Workspace: p.WorkDir,
		Version:   p.Schema.Version,
	}
	err = r.Download(p.Schema.Resources)
	if err != nil {
		return err
	}
	// 获取构建插件所用的协议版本
	schemaVersion, err := getSchemaVersion(p.GitInstance, p.Path)
	if err != nil {
		return err
	}
	err = p.handlePostLoad()
	if err != nil {
		return err
	}
	err = writeDoneFile(doneFile, LoadDoneInfo{
		SchemaVersion: schemaVersion,
		Version:       p.Schema.Version,
		LoadTime:      time.Now(),
	})
	return err
}

func (p *tCLIPlugin) handlePreLoad() error {
	preloadCommands := p.Schema.SelectCommands(p.WorkDir, p.Schema.PreLoad)
	if len(preloadCommands) == 0 {
		return nil
	}
	ioc.Log.Infof("正在执行初始化命令...")
	err := runCommands(preloadCommands, p.Envs, nil)
	if err != nil {
		return fmt.Errorf("初始化命令执行失败，无法继续加载：%w", err)
	}
	return nil
}

func (p *tCLIPlugin) handlePostLoad() error {
	postLoadCommands := p.Schema.SelectCommands(p.WorkDir, p.Schema.PostLoad)
	if len(postLoadCommands) == 0 {
		return nil
	}
	ioc.Log.Infof("正在执行加载后命令...")
	err := runCommands(postLoadCommands, p.Envs, nil)
	if err != nil {
		return fmt.Errorf("加载后命令执行失败，请排查：%w", err)
	}
	return nil
}

func (p *tCLIPlugin) buildDep(depPath string, gitInstance *git.Git, binPath string) error {
	rootPath := gitInstance.RootPath()
	absDepPath := strings.TrimPrefix(depPath, "/")
	absDepPath = filepath.Join(rootPath, absDepPath)
	depPlugin, err := newTCLIPlugin(p.Cli, absDepPath)
	if err != nil {
		if err == ErrUnSupported {
			ioc.Log.Debugf("当前环境不支持此依赖")
			return nil
		}
		return fmt.Errorf("解析依赖插件 [%s] 失败，该插件路径为：%s，%w", depPath, absDepPath, err)
	}
	// 添加依赖调度指令
	// 这里可以对依赖进行加载，也即，加载插件时，会同时加载所有插件的依赖
	// 但并非所有依赖都可能被使用，例如代码扫描，依赖了各类语言的扫描插件，但对实际的用户来说可能只会用到其中一两种
	// 所以这里改用完全懒加载模式，只构建运行时索引，在真正用到时才会加载
	_, err = linker.New(depPlugin.Info().Name, binPath, execDepPluginCommand(absDepPath), linker.OverrideAlways)
	if err != nil {
		return fmt.Errorf("连接依赖插件 [%s] 失败，该插件路径为：%s，%w", depPath, absDepPath, err)
	}
	return nil
}

func writeDoneFile(doneFile string, info LoadDoneInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(doneFile, data, os.ModePerm)
}

// execDepPluginCommand 被依赖插件的执行命令
func execDepPluginCommand(target string) string {
	return fmt.Sprintf(`%s exec "%s"`, config.AliasName, target)
}

// depBinPath 获取工具依赖的 bin 目录，将注入到环境变量，以确保工具运行时能正确找到自己的依赖
func depBinPath(pluginWorkDir string) string {
	return filepath.Join(pluginWorkDir, config.DepBinPath)
}

// runCommands 执行命令集
func runCommands(commands []string, envs, args []string) error {
	for _, command := range commands {
		err := runCommand(command, envs, args)
		if err != nil {
			return err
		}
	}
	return nil
}

// runCommand 执行单句命令
func runCommand(command string, envs, args []string) error {
	shell := command + " " + cli.QuoteCommand(args)
	ioc.Log.Infof("实际执行命令：%s", shell)
	// 继承IO并执行
	return cli.Local().RunParamsInherit(context.TODO(), cli.Params{
		Shell: shell,
		Env:   envs,
	})
}

func (p *tCLIPlugin) prepareEnvs() error {
	if len(p.Envs) > 0 {
		return nil
	}
	binPath, err := getBinPath()
	if err != nil {
		return err
	}
	envs := getEnvs()
	depPath := filepath.Join(p.WorkDir, config.DepBinPath)
	t2BinPath := filepath.Join(config.AppConfDir, config.BinPath)
	newPaths := append([]string{}, binPath, depPath, t2BinPath)
	envPaths := path.GetEnvPaths(true, newPaths...)
	envs[config.PATH] = envPaths
	// 兼容历史依赖此环境变量的 feflow 插件
	// 主要是 code-style-js 依赖了此变量
	if !p.Schema.Runtime {
		envs[config.FeflowPluginPath] = p.WorkDir
	}
	envs[config.TCLIPluginPath] = p.WorkDir
	envs[config.TCLICommandPath] = p.GitInstance.RootPath()
	envs[config.TCLIVersion] = config.Version
	envs[config.TCLIFrom] = p.Info().Name
	for key, value := range envs {
		p.Envs = append(p.Envs, fmt.Sprintf("%s=%s", key, value))
	}
	return nil
}

func getEnvs() map[string]string {
	envMap := map[string]string{}
	for _, env := range config.Envs {
		items := strings.Split(env, "=")
		key := items[0]
		value := strings.Join(items[1:], "=")
		// 背景：部分平台，PATH可能会表现为 Path，path，PATH 等无法预估的形式
		// 导致环境变量处理不符合预期，所以应该统一将它标准为 PATH
		if strings.EqualFold(key, config.PATH) {
			envMap[strings.ToUpper(key)] = value
		} else {
			envMap[key] = value
		}
	}
	return envMap
}

func getBinPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		execPath = os.Args[0]
	}
	// TODO(blurooochen): 减少不同主程序相互干扰的情况
	currentVersion := config.Version
	binPath := filepath.Join(config.TempDir, config.DepBinPath, currentVersion)
	// 注册主程序命令，保证任何位置启动主程序都可以完整运行
	_, err = linker.New(config.AliasName, binPath, execPath, linker.OverrideAlways)
	if err != nil {
		return "", err
	}
	return binPath, nil
}
