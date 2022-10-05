package errs

const (
	// CodeSuccess 正常
	CodeSuccess = 0
	// CodeUnknown 未知错误码
	CodeUnknown = 1
	// CodeParamInvalid 使用命令行的参数不正确
	CodeParamInvalid = 63
	// CodeRuntimeConfigUnready 运行时配置未就绪
	CodeRuntimeConfigUnready = 64

	// CodeRepoOpenFail 仓库打开失败
	CodeRepoOpenFail = 70
	// CodeRepoCloneFail 仓库克隆失败
	CodeRepoCloneFail = 71
	// CodeRepoPullFail 仓库拉取失败
	CodeRepoPullFail = 72

	// CodeFileWalkError 文件夹遍历错误
	CodeFileWalkError = 80
	// CodePluginLoadFail 插件加载失败
	CodePluginLoadFail = 81
	// CodePluginEnvUnready 插件环境未就绪
	CodePluginEnvUnready = 82
	// CodePluginExecError 插件执行错误
	CodePluginExecError = 83
	// CodePluginArgsParseError 插件参数解析失败
	CodePluginArgsParseError = 84
	// CodeShellTerminalNotFound 脚本终端未找到
	CodeShellTerminalNotFound = 85
)
