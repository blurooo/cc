package config

// Flags 从命令行参数收集到的配置
var Flags = &struct {
	// Envs 环境变量配置
	Envs []string
	// Inquire 是否询问模式
	Inquire bool
	// Global 全局模式
	Global bool

	// EventType 监听事件类型
	EventType string
	// EventInfo 监听事件荷载数据 json格式
	EventInfo string
}{
	Inquire: true,
}
