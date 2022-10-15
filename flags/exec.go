package flags

import (
	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/config"
	"github.com/spf13/cobra"
)

func GetExecCommand(app config.Application) *cobra.Command {
	// ExecCommand 负责调起 tc 主程序执行依赖插件（包含插件的资源加载、入口解析、数据上报等逻辑），没必要对外开放，所以进行隐藏
	return &cobra.Command{
		Use:                "__exec <plugin> ...",
		Short:              "exec the special plugin file",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		Hidden:             true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			path := args[0]
			args = args[1:]
			cc := &command.CobraCommands{App: app}
			return cc.ExecFile(path, args)
		},
	}

}
