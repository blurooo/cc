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
	// CodeFileOperationFail 文件操作失败
	CodeFileOperationFail = 65
)
