package main

import (
	"fmt"
	"os"
	"time"

	"tencent2/tools/dev_tools/t2cli/daemon"
	"tencent2/tools/dev_tools/t2cli/exit"
)

func main() {
	asyncProcess := &daemon.AsyncProcess{
		Name:      "test",
		Singleton: true,
		Version:   "test",
		Args:      []string{"hello"},
	}
	info, err := asyncProcess.Start()
	if err != nil {
		panic(err)
	}
	if info != nil {
		// 执行当前主进程任务
		fmt.Println("hello main process !!", os.Args)
		return
	}
	// 执行当前主进程任务
	fmt.Println("hello async process !!", os.Args)
	// 执行异步进程任务
	for i := 0; i < 10; i++ {
		fmt.Println(os.Getpid(), i)
		time.Sleep(time.Second * 1)
	}
	fmt.Printf("启动成功: %d, %s\n", os.Getpid(), time.Now())
	// 使用优雅退出收尾
	exit.Gracefully(0)
}
