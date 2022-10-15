package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var configFlags = struct {
	setters []string
	getters []string
	list    bool
}{}

var updateFlags = struct {
	All bool
}{}

func (f *Flags) getConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "config",
		Short:             "配置域相关能力，包括工具版本、程序运行参数等",
		ValidArgsFunction: EnableFlagsCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFlags.list {
				return f.handleList()
			}
			if len(configFlags.getters) > 0 {
				return f.handleGetter()
			}
			if len(configFlags.setters) > 0 {
				return f.handleSetter()
			}
			return cmd.Help()
		},
	}
}

var initCommand = &cobra.Command{
	Use:    "_init",
	Short:  "初始化程序",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// execPath, err := os.Executable()
		// if err != nil {
		// 	return fmt.Errorf("获取当前程序的执行路径失败：%w", err)
		// }
		// _, err = tc.Init(execPath)
		// return err
		return nil
	},
}

var updateCommand = &cobra.Command{
	Use:               "update",
	Short:             "更新所有工具版本",
	ValidArgsFunction: EnableFlagsCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		// return tc.UpdateTools(tc.UpdateStrategy{All: updateFlags.All})
		return nil
	},
}

func setUpdateFlags() {
	updateCommand.Flags().BoolVar(&updateFlags.All, "all", false, "update all")
}

func (f *Flags) setConfigFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&configFlags.getters, "get", nil, "获取程序配置，支持多条")
	cmd.Flags().StringArrayVar(&configFlags.setters, "set", nil, "设置程序运行时参数，支持多条")
	cmd.Flags().BoolVar(&configFlags.list, "list", false, "获取配置列表")

	_ = cmd.RegisterFlagCompletionFunc("get", f.configCompletion)
	_ = cmd.RegisterFlagCompletionFunc("set", f.configCompletion)
}

func (f *Flags) configCompletion(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	// 动态补全可用的配置列表
	configs, err := f.Configurator.ListUsableConfigs()
	if err != nil {
		return nil, cobra.ShellCompDirectiveDefault
	}
	completionArgs := make([]string, 0, len(configs))
	for _, c := range configs {
		completionArgs = append(completionArgs, fmt.Sprintf("%s\t%s", c.Key, c.Comment))
	}
	return completionArgs, cobra.ShellCompDirectiveDefault
}

func (f *Flags) handleSetter() error {
	for _, setter := range configFlags.setters {
		items := strings.Split(setter, "=")
		key := items[0]
		value := strings.Join(items[1:], "=")
		err := f.Configurator.SetConfig(key, value)
		if err != nil {
			return fmt.Errorf("配置参数 %s 失败：%w", setter, err)
		}
	}
	return nil
}

func (f *Flags) handleGetter() error {
	showValueOnly := len(configFlags.getters) == 1
	for _, getter := range configFlags.getters {
		v, err := f.Configurator.GetConfig(getter)
		if err != nil {
			return fmt.Errorf("获取配置 %s 失败：%w", getter, err)
		}
		if showValueOnly {
			fmt.Println(v)
		} else {
			fmt.Printf("%s=%s\n", getter, v)
		}
	}
	return nil
}

func (f *Flags) handleList() error {
	items, err := f.Configurator.ListUsedConfigs()
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Println(item)
	}
	return nil
}

// EnableFlagsCompletion 开启 flags 的自动补全机制
func EnableFlagsCompletion(cmd *cobra.Command,
	_ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var args []string
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		args = append(args, fmt.Sprintf("--%s\t%s", flag.Name, flag.Usage))
		if flag.Shorthand != "" {
			args = append(args, fmt.Sprintf("-%s\t%s", flag.Shorthand, flag.Usage))
		}
	})
	usage := "get help for command"
	args = append(args, fmt.Sprintf("--help\t%s", usage), fmt.Sprintf("-h\t%s", usage))
	return args, cobra.ShellCompDirectiveDefault
}
