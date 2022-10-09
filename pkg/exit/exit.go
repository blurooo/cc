// Package exit 聚焦于进程退出的收尾工作
// 通过 listen 监听进程退出事件时，将只会在收到结束信号，例如 kill ctrl + c 时，才会进入收尾逻辑
// 如果希望进程正常退出时也进行收尾，可以通过 exit.Gracefully 来结束进程生命周期
package exit

import (
	"context"
	"os"
)

// Handle 优雅退出处理函数
type Handle func(ctx context.Context, signal os.Signal)

// Exit 退出标准
type Exit interface {
	// Listen 监听所有可捕获的退出信号
	Listen(handle Handle)
	// ListenSignal 监听某个可捕获的退出信号
	ListenSignal(signal os.Signal, handle Handle)
}

// ProcessExit 进程退出接口
type ProcessExit interface {
	Exit(code int)
}
