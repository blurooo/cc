package flags

import (
	"fmt"

	"github.com/blurooo/cc/config"
	"github.com/spf13/cobra"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/tc"
)

var installFlags = struct {
	list bool
}{}

// 负责调起 tc 主程序执行依赖插件（包含插件的资源加载、入口解析、数据上报等逻辑），没必要对外开放，所以进行隐藏
func getInstallCommand(config config.ApplicationConfig) *cobra.Command {
	return &cobra.Command{
		Use:           "install <name> [--list]",
		Short:         "安装某个命令到系统中，后续将直接使用命令进行调用",
		SilenceErrors: true,
		SilenceUsage:  true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string,
			toComplete string) ([]string, cobra.ShellCompDirective) {
			completions, directive := EnableFlagsCompletion(cmd, args, toComplete)
			commands, _ := tc.InstallableList()
			for _, command := range commands {
				completions = append(completions, fmt.Sprintf("%s\t%s", command.Name, command.Desc))
			}
			return completions, directive
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if installFlags.list {
				return handleInstallList(config)
			}
			return handleInstall(config, args)
		},
	}
}

func handleInstallList(config config.ApplicationConfig) error {
	commands, err := tc.InstallableList()
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		config.Logger.Infof("暂无可供安装到系统的命令")
		return nil
	}
	firstCommand := commands[0]
	name := firstCommand.Plugin.Name()
	config.Logger.Infof("查找到以下可被安装的插件列表，例如：%s install %s 即可安装 %s", config.Name, name, name)
	for i, cmd := range commands {
		config.Logger.Infof("%d. name: %s, version: %s, schema-path: %s", i+1, cmd.Plugin.Name(), cmd.Plugin.Version(), cmd.AbsPath)
	}
	return nil
}

func handleInstall(config config.ApplicationConfig, args []string) error {
	if len(args) == 0 {
		config.Logger.Warnf("参数不全，请提供需要安装的指令名")
		return nil
	}
	commands, err := tc.InstallableList()
	if err != nil {
		return err
	}
	for _, name := range args {
		err = install(config, commands, name)
		if err != nil {
			return fmt.Errorf("安装失败，%w", err)
		}
	}
	return nil
}

func install(config config.ApplicationConfig, commands []command.Node, name string) error {
	absPath := getAbsPath(commands, name)
	if absPath == "" {
		config.Logger.Warnf("[%s] 未找到，请确认名称是否正确", name)
		return nil
	}
	config.Logger.Infof("正在安装 %s...", name)
	err := tc.InstallFile(absPath)
	if err != nil {
		return err
	}
	return nil
}

func getAbsPath(commands []command.Node, name string) string {
	for _, cmd := range commands {
		if cmd.Plugin.Name() == name {
			return cmd.AbsPath
		}
	}
	return ""
}

func setInstallFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&installFlags.list, "list", false, "获取可安装命令列表")
}

func AddInstallCommand(rc *cobra.Command, config config.ApplicationConfig) {
	installCommand := getInstallCommand(config)
	setInstallFlags(installCommand)
	addToRootCmd(rc, installCommand)
}
