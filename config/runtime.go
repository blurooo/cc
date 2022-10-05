package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blurooo/cc/errs"
	"github.com/blurooo/cc/ioc"
	"github.com/blurooo/cc/util/log"
	"tencent2/tools/dev_tools/t2cli/common/cfile"
	"tencent2/tools/dev_tools/t2cli/report"
)

// AppName 应用名
const AppName = "tc"

// AliasName 别名
// TODO(blurooochen): remove tc
const AliasName = "metax"

const (
	// repoStageDirName 指令仓库统一保存路径
	repoStageDirName = "repo"
	// logDirName 日志文件名
	logDirName = "log"
	// daemonDirName 守护文件夹名称
	daemonDirName = "daemon"
	// pluginDirName 插件资源工作目录
	pluginDirName = "plugins"
	// tempDirName 其它缓存目录
	tempDirName = "temp"
	// binDirName 可执行文件存放目录
	binDirName = "bin"
	// resourceDirName 资源存放目录
	resourceDirName = "resource"
)

var (
	// Debug 调试
	Debug bool
	// Version 工具版本
	Version = "test"
	// AppConfDir 程序的配置目录
	AppConfDir string
	// LogDir 日志保存目录
	LogDir string
	// DaemonDir 守护进程文件夹
	DaemonDir string
	// RepoStashDir 仓库暂存目录
	RepoStashDir string
	// PluginDir 插件所在目录
	PluginDir string
	// TempDir 缓存目录
	TempDir string
	// BinDir 可执行文件存放目录
	BinDir string
	// ResourceDir 资源目录
	ResourceDir string
	// AppConfigFile 应用配置文件
	AppConfigFile string
)

// Envs 环境变量
var Envs []string

// unsetEnvs 管控环境变量，移除子进程的代理能力
var unsetEnvs = []string{"http_proxy", "https_proxy", "all_proxy", "no_proxy"}

// PersistentConfig 持久化配置
type PersistentConfig struct {
	Update  Update  `ini:"update"`
	Command Command `ini:"command"`
}

// Update 更新策略配置
type Update struct {
	Always bool `ini:"always" comment:"自动进行版本更新"`
}

// Command 指令集配置
type Command struct {
	Repo string `ini:"repo" comment:"自定义指令仓库，例如 https://xx.git"`
	Path string `ini:"path" comment:"自定义指令目录，将动态指令集指向本地某个路径"`
}

func initRuntime() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return errs.NewProcessErrorWithCode(err, errs.CodeRuntimeConfigUnready)
	}
	AppConfDir = cfile.Resolve(home, fmt.Sprintf(".%s", AppName), true)
	LogDir = cfile.Resolve(AppConfDir, logDirName, true)
	DaemonDir = cfile.Resolve(AppConfDir, daemonDirName, true)
	AppConfigFile = filepath.Join(AppConfDir, "config.ini")
	RepoStashDir = cfile.Resolve(AppConfDir, repoStageDirName, true)
	PluginDir = cfile.Resolve(AppConfDir, pluginDirName, true)
	TempDir = cfile.Resolve(AppConfDir, tempDirName, true)
	BinDir = cfile.Resolve(AppConfDir, binDirName, false)
	ResourceDir = cfile.Resolve(AppConfDir, resourceDirName, false)
	return nil
}

func initEnvs() {
	envs := os.Environ()
	for _, env := range envs {
		if isUnsetEnv(env) {
			continue
		}
		Envs = append(Envs, env)
	}
}

func isUnsetEnv(env string) bool {
	for _, unsetEnv := range unsetEnvs {
		if strings.HasPrefix(env, unsetEnv+"=") {
			return true
		}
	}
	return false
}

func initIOC() {
	ioc.Reporter = report.NewReporter()
	ioc.Log = log.Logrus(log.Param{
		Debug: Debug,
	})
}
