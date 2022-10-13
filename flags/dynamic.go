package flags

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/pkg/helper"
	"github.com/blurooo/cc/tc"
	"github.com/spf13/cobra"
)

// cobraRunner cobra 的运行函数
type cobraRunner func(cmd *cobra.Command, args []string) error

// cobraHelper cobra 的帮助函数
type cobraHelper func(cmd *cobra.Command, args []string)

func (f *Flags) registerDynamicCommands(rc *cobra.Command) error {
	// 获取动态命令
	nodes, err := tc.Nodes()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		cmd := f.toCommand(node)
		if cmd != nil {
			rc.AddCommand(cmd)
		}
	}
	return nil
}

func (f *Flags) toCommand(node command.Node) *cobra.Command {
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
		return f.toSubCommand(node)
	}
	var cmd *cobra.Command
	if parentCmd == nil {
		parentCmd = defaultCobraCommand(node)
		cmd = parentCmd
	}
	for _, child := range node.Children {
		parentCmd.AddCommand(f.toCommand(child))
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

func (f *Flags) toSubCommand(node command.Node) *cobra.Command {
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
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(f.toCobraHelper(node, defaultHelpFunc))
	return cmd
}

func toCobraRunner(node command.Node) cobraRunner {
	return func(cmd *cobra.Command, args []string) error {
		return tc.ExecNode(node, args)
	}
}

func (f *Flags) toCobraHelper(node command.Node, defaultHelper func(cmd *cobra.Command, args []string)) cobraHelper {
	return func(cmd *cobra.Command, args []string) {
		helpFile := filepath.Join(node.Dir, fmt.Sprintf("%s.md", node.Name))
		var err error
		if _, err = os.Stat(helpFile); err != nil {
			if defaultHelper == nil {
				err = tc.ExecNode(node, []string{"--help"})
			} else {
				defaultHelper(cmd, args)
			}
		} else {
			err = helper.Help(helpFile)
		}
		if err != nil {
			f.ApplicationConfig.Logger.Errorf("get help failed: %v", err)
		}
	}
}
