package config

// 环境变量相关配置
const (
	// TCLIVersion tcli版本
	TCLIVersion = "TCLI_VERSION"
	// TCLITraceID 跟踪ID
	TCLITraceID = "TCLI_TRACE_ID"
	// TCLIFrom 调用方
	TCLIFrom = "TCLI_FROM"
	// TCLIPluginPath 插件的工作目录，同 FEF_PLUGIN_PATH
	TCLIPluginPath = "TCLI_PLUGIN_PATH"
	// TCLICommandPath 指令源路径
	TCLICommandPath = "TCLI_COMMAND_PATH"

	// FeflowPluginPath feflow 插件路径在环境变量中的键名
	FeflowPluginPath = "FEF_PLUGIN_PATH"

	// PATH 环境变量PATH键
	PATH = "PATH"
)

// 仓库相关配置
const (
	// CommandRepoURL 全局指令仓库
	CommandRepoURL = ""

	// CommandDir 指令目录
	CommandDir = "cmd"

	// InstallDir 可安装指令目录
	InstallDir = "install"

	// DefaultVersion 默认版本
	DefaultVersion = "0.0.0"

	// DoneFile 完成标志文件
	DoneFile = ".done"

	// DepBinPath 依赖的 bin 目录
	DepBinPath = ".bin"

	// BinPath tc 统一 bin 目录
	BinPath = "bin"
)

const (
	// RepoAuthUser 仓库统一认证用户
	RepoAuthUser = ""
	// RepoAuthPwd 仓库统一认证密码
	RepoAuthPwd = ""
)

const (
	// EnvUpdateSelf 环境变量，更新自身
	EnvUpdateSelf = "TCLI_UPDATE_SELF"
)

const (
	// SceneT2Plugins 插件场景
	SceneT2Plugins = "t2_plugins"
	// SceneT2Dependence 插件依赖场景
	SceneT2Dependence = "t2_dependence"
	// SceneT2Helper 帮助场景
	SceneT2Helper = "t2_helper"
)

const (
	// True 字符串 true
	True = "true"
)
