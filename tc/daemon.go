package tc

import (
	"os"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/ioc"
)

// StartDaemon 启用守护进程
func StartDaemon() error {
	// 输出重定向到标准输出，预期是异步进程的日志文件
	ioc.Log.SetOutput(os.Stdout)
	// 避免继承此标志，导致无法自动更新工具自身
	if err := os.Unsetenv(config.EnvUpdateSelf); err != nil {
		ioc.Log.Infof("移除更新标志失败：%v", err)
	}
	c := cron.New()

	// 开始注册任务
	_, err := c.AddFunc("@every 1m", autoUpdate)
	if err != nil {
		return err
	}

	c.Run()
	return nil
}

func autoUpdate() {
	c, err := load()
	if err != nil {
		ioc.Log.Errorf("获取配置出错：%v", err)
		return
	}
	if !c.Update.Always {
		ioc.Log.Info("暂未启用自动更新，当前任务跳过")
		return
	}
	err = UpdateTools(UpdateStrategy{All: false})
	if err != nil {
		ioc.Log.Error(err)
	}
}
