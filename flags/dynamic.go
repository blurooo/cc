package flags

import (
	"fmt"
	"path/filepath"

	"github.com/blurooo/cc/pkg/helper"
	"github.com/spf13/cobra"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
	"tencent2/tools/dev_tools/t2cli/report"
	"tencent2/tools/dev_tools/t2cli/schemas/input"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/ioc"
	"github.com/blurooo/cc/tc"
	"github.com/blurooo/cc/util/reporter"
)

// cobraRunner cobra 的运行函数
type cobraRunner func(cmd *cobra.Command, args []string) error

// cobraHelper cobra 的帮助函数
type cobraHelper func(cmd *cobra.Command, args []string)

func registerDynamicCommands() error {
	// 获取动态命令
	nodes, err := tc.Nodes()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		cmd := toCommand(node)
		if cmd != nil {
			rootCmd.AddCommand(cmd)
		}
	}
	return nil
}

func toCommand(node command.Node) *cobra.Command {
	fullName := node.FullName(nameSplit)
	var parentCmd *cobra.Command
	// 如果存在原生指令集，则直接注册到该指令集内，完成融合
	if nativeCommand, ok := commandIndex[fullName]; ok {
		if nativeCommand.HasSubCommands() {
			parentCmd = nativeCommand
		} else {
			return nil
		}
	}
	if node.IsLeaf {
		return toSubCommand(node)
	}
	var cmd *cobra.Command
	if parentCmd == nil {
		parentCmd = defaultCobraCommand(node)
		cmd = parentCmd
	}
	for _, child := range node.Children {
		parentCmd.AddCommand(toCommand(child))
	}
	return cmd
}

func defaultCobraCommand(node command.Node) *cobra.Command {
	return &cobra.Command{
		Use:   node.Name,
		Short: node.Desc,
		Long:  node.Desc,
		// flag 解析由动态指令决定
		DisableFlagParsing: true,
	}
}

func toSubCommand(node command.Node) *cobra.Command {
	// 实时构建 cobra 子命令
	cmd := &cobra.Command{
		Use:   node.Name,
		Short: node.Desc,
		Long:  node.Desc,
		RunE:  toCobraRunner(node),
		// 动态命令不要自行打印错误造成干扰（help）
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	if len(node.Plugin.Scenes()) == 0 {
		// 未配置场景化能力的话，参数将直接进行透传，所以关闭 flag 的解析能力
		cmd.DisableFlagParsing = true
	}
	inputs := make(map[string]interface{})
	for _, scene := range node.Plugin.Scenes() {
		registerFlags(cmd, inputs, scene.Inputs)
	}
	if len(inputs) > 0 {
		cmd.RunE = toCobraRunnerWithInputs(node, inputs)
	}
	defaultHelpFunc := cmd.HelpFunc()
	// 如果配置了场景化输入参数，则帮助信息获取可以采用 cobra 的默认策略
	// 否则采用 --help 透传，
	if len(inputs) == 0 {
		defaultHelpFunc = nil
	}
	cmd.SetHelpFunc(toCobraHelper(node, defaultHelpFunc))
	return cmd
}

func registerFlags(cmd *cobra.Command, inputs map[string]interface{}, components input.Components) {
	// 组件的 prop 映射为 --${prop}
	for _, component := range components {
		switch component.GetTypeInfo().Type {
		case input.BOOL:
			inputs[component.GetProp()] = registerBoolFlag(cmd, component)
		default:
			inputs[component.GetProp()] = registerStringFlag(cmd, component)
		}
	}
}

func registerBoolFlag(cmd *cobra.Command, component input.Component) *bool {
	var val bool
	var defaultValue bool
	if defaultIsTrue(component) {
		defaultValue = true
	}
	cmd.Flags().BoolVar(&val, component.GetProp(), defaultValue, component.GetUsage())
	return &val
}

func defaultIsTrue(component input.Component) bool {
	return component.GetDefault() == config.True
}

func registerStringFlag(cmd *cobra.Command, component input.Component) *string {
	var val string
	cmd.Flags().StringVar(&val, component.GetProp(), component.GetDefault(), component.GetUsage())
	return &val
}

func toCobraRunner(node command.Node) cobraRunner {
	return func(cmd *cobra.Command, args []string) error {
		entry := genEntryFromCommand(node, args)
		entry.Scene = config.SceneT2Plugins
		err := tc.ExecNode(node, args)
		entry.End(err)
		ioc.Reporter.Add(entry)
		return err
	}
}

func toCobraRunnerWithInputs(node command.Node, inputs map[string]interface{}) cobraRunner {
	return func(cmd *cobra.Command, args []string) error {
		execArgs := make([]string, 0, len(inputs)+len(args))
		for k, v := range inputs {
			if v == nil {
				continue
			}
			execArgs = append(execArgs, fmt.Sprintf("%s=%s", k, getInputValue(v)))
		}
		for i, arg := range args {
			execArgs = append(execArgs, fmt.Sprintf("%d=%s", i, arg))
		}
		entry := genEntryFromCommand(node, args)
		entry.Scene = config.SceneT2Plugins
		err := tc.ExecNode(node, execArgs)
		entry.End(err)
		ioc.Reporter.Add(entry)
		return err
	}
}

func getInputValue(iVal interface{}) string {
	var value string
	// 目前只处理 *string 和 *bool 两种类型
	switch v := iVal.(type) {
	case *bool:
		value = fmt.Sprintf("%v", *v)
	case *string:
		value = *v
	default:
		ioc.Log.Debugf("不支持的输入类型：%v", iVal)
		value = ""
	}
	return value
}

func toCobraHelper(node command.Node, defaultHelper func(cmd *cobra.Command, args []string)) cobraHelper {
	return func(cmd *cobra.Command, args []string) {
		entry := genEntryFromCommand(node, args)
		entry.Scene = config.SceneT2Helper
		var err error
		helpFile := filepath.Join(node.Dir, fmt.Sprintf("%s.md", node.Name))
		if !cfile.Exist(helpFile) {
			if defaultHelper == nil {
				err = tc.ExecNode(node, []string{"--help"})
			} else {
				defaultHelper(cmd, args)
			}
		} else {
			err = helper.Help(helpFile)
		}
		entry.End(err)
		ioc.Reporter.Add(entry)
		if err != nil {
			ioc.Log.Errorf("获取帮助信息失败：%v", err)
		}
	}
}

func genEntryFromCommand(node command.Node, args []string) report.Entry {
	entry := reporter.NewEntry(args)
	entry.Command = node.FullName(".")
	info := node.Plugin.Info()
	entry.Version = info.Version
	return entry
}
