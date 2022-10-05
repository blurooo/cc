package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"

	"tencent2/tools/dev_tools/t2cli/exit"
	"tencent2/tools/dev_tools/t2cli/report/data"
	"tencent2/tools/dev_tools/t2cli/t2/cmd/flags"
	"github.com/blurooo/cc/errs"
	"github.com/blurooo/cc/ioc"
)

func main() {
	handleError(execute())
}

func execute() error {
	exit.New(time.Second * 2).Listen(report)
	return flags.Execute()
}

func report(ctx context.Context, _ os.Signal) {
	err := ioc.Reporter.Report(ctx)
	if err != nil {
		ioc.Log.Debugf("数据上报异常：%v", err)
	}
}

func handleError(err error) {
	if err == nil {
		exit.Gracefully(errs.CodeSuccess)
	}
	ioc.Log.Errorf("%s [TraceID: %s]", err, data.TraceID())
	var eErr *exec.ExitError
	// 优先继承进程退出码
	if ok := errors.As(err, &eErr); ok {
		exit.Gracefully(eErr.ExitCode())
	} else {
		exit.Gracefully(errs.CodeUnknown)
	}
}
