package flags

import (
	"github.com/spf13/cobra"

	"github.com/blurooo/cc/tc"
)

// 负责调起 tc 主程序执行依赖插件（包含插件的资源加载、入口解析、数据上报等逻辑），没必要对外开放，所以进行隐藏
var execCommand = &cobra.Command{
	Use:                "exec <plugin> ...",
	Short:              "执行指定的指令协议文件",
	DisableFlagParsing: true,
	SilenceErrors:      true,
	SilenceUsage:       true,
	Hidden:             true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		pluginPath := args[0]
		args = args[1:]
		return tc.ExecFile(pluginPath, args)
	},
}

func registerExecCmd() {
	addToRootCmd(execCommand)
}
