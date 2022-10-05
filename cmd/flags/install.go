package flags

import (
	"fmt"

	"github.com/spf13/cobra"

	"tencent2/tools/dev_tools/t2cli/common/flags"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/ioc"
	"github.com/blurooo/cc/tc"
)

var installFlags = struct {
	list bool
}{}

// 负责调起 tc 主程序执行依赖插件（包含插件的资源加载、入口解析、数据上报等逻辑），没必要对外开放，所以进行隐藏
var installCommand = &cobra.Command{
	Use:           "install <name> [--list]",
	Short:         "安装某个命令到系统中，后续将直接使用命令进行调用",
	SilenceErrors: true,
	SilenceUsage:  true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string,
		toComplete string) ([]string, cobra.ShellCompDirective) {
		completions, directive := flags.EnableFlagsCompletion(cmd, args, toComplete)
		commands, _ := tc.InstallableList()
		for _, command := range commands {
			completions = append(completions, fmt.Sprintf("%s\t%s", command.Name, command.Desc))
		}
		return completions, directive
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if installFlags.list {
			return handleInstallList()
		}
		return handleInstall(args)
	},
}

func handleInstallList() error {
	commands, err := tc.InstallableList()
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		ioc.Log.Infof("暂无可供安装到系统的命令")
		return nil
	}
	firstCommand := commands[0]
	name := firstCommand.Plugin.Info().Name
	ioc.Log.Infof("查找到以下可被安装的插件列表，例如：%s install %s 即可安装 %s", config.AliasName, name, name)
	for i, cmd := range commands {
		info := cmd.Plugin.Info()
		ioc.Log.Infof("%d. name: %s, version: %s, schema-path: %s", i+1, info.Name, info.Version, cmd.AbsPath)
	}
	return nil
}

func handleInstall(args []string) error {
	if len(args) == 0 {
		ioc.Log.Warnf("参数不全，请提供需要安装的指令名")
		return nil
	}
	commands, err := tc.InstallableList()
	if err != nil {
		return err
	}
	for _, name := range args {
		err = install(commands, name)
		if err != nil {
			return fmt.Errorf("安装失败，%w", err)
		}
	}
	return nil
}

func install(commands []command.Node, name string) error {
	absPath := getAbsPath(commands, name)
	if absPath == "" {
		ioc.Log.Warnf("[%s] 未找到，请确认名称是否正确", name)
		return nil
	}
	ioc.Log.Infof("正在安装 %s...", name)
	err := tc.InstallFile(absPath)
	if err != nil {
		return err
	}
	return nil
}

func getAbsPath(commands []command.Node, name string) string {
	for _, cmd := range commands {
		info := cmd.Plugin.Info()
		if info.Name == name {
			return cmd.AbsPath
		}
	}
	return ""
}

func registerInstallCmd() {
	fs := installCommand.Flags()
	fs.BoolVar(&installFlags.list, "list", false, "获取可安装命令列表")

	addToRootCmd(installCommand)
}
