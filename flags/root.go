// Package flags 命令行参数相关处理逻辑
package flags

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/log"
	"github.com/blurooo/cc/pkg/daemon"
	"github.com/spf13/cobra"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/errs"
)

const nameSplit = "."

const daemonName = "cc.daemon"

var commandIndex = map[string]*cobra.Command{}

type Flags struct {
	App           config.Application
	CobraCommands command.CobraCommands
}

func (f *Flags) Execute() error {
	rc := &cobra.Command{
		Use:               f.App.Name,
		Short:             f.App.Desc,
		Long:              f.App.Desc,
		ValidArgsFunction: EnableFlagsCompletion,
		Version:           fmt.Sprintf("%s (%s.%s)", f.App.Version, runtime.GOOS, runtime.GOARCH),
	}
	// 注册内建命令
	f.initBuiltinCmd(rc)
	cmd, _, err := rc.Find(os.Args[1:])
	// 仅在没有命中内置子命令时，注册动态指令，提高内置指令的运行性能
	// 如果只命中根命令也会启用动态指令，便于打印动态指令的帮助信息
	if ok := f.startDaemon(cmd); ok {
		return nil
	}
	// 处理动态命令
	if err = f.initDynamicCommands(rc, cmd, err); err != nil {
		return err
	}
	// 成功处理自动补全时，不需要继续往下走
	if ok := f.handleComplete(rc); ok {
		return nil
	}
	return handleKnownError(rc.Execute())
}

func (f *Flags) handleComplete(rc *cobra.Command) bool {
	completeCmd := &cobra.Command{
		Use:                   cobra.ShellCompRequestCmd,
		Aliases:               []string{cobra.ShellCompNoDescRequestCmd},
		DisableFlagsInUseLine: true,
		Hidden:                true,
		DisableFlagParsing:    true,
	}
	rc.AddCommand(completeCmd)
	defer rc.RemoveCommand(completeCmd)
	cmd, args, err := rc.Find(os.Args[1:])
	if err != nil {
		return false
	}
	if cmd != completeCmd {
		return false
	}
	cmd, args, err = rc.Find(args)
	if err != nil {
		return false
	}
	// 识别动态子命令
	if !isDynamicCommand(cmd) {
		return false
	}
	err = f.handleDynamicComplete(cmd, args)
	return err == nil
}

func (f *Flags) handleDynamicComplete(cmd *cobra.Command, args []string) error {
	// 移除掉所有日志相关的干扰性输出
	f.App.Logger = log.Discard
	args = append([]string{os.Args[1]}, args...)
	if cmd.RunE != nil {
		return cmd.RunE(cmd, args)
	}
	cmd.Run(cmd, args)
	return nil
}

func isDynamicCommand(cmd *cobra.Command) bool {
	return cmd.DisableFlagParsing && isFinalCommand(cmd)
}

func isFinalCommand(cmd *cobra.Command) bool {
	return cmd.RunE != nil || cmd.Run != nil
}

func handleKnownError(err error) error {
	if err == nil {
		return nil
	}
	if isUnknownCommandError(err) {
		return nil
	}
	return err
}

func isUnknownCommandError(err error) bool {
	return strings.HasPrefix(err.Error(), "unknown command")
}

func addToRootCmd(rc *cobra.Command, cmd *cobra.Command) {
	rc.AddCommand(cmd)
	// 以平铺的方式构建原生指令索引
	// exec
	// config
	// config.init
	// config.update-tools
	// ...
	// 方便在于动态指令融合时，被查找复用指令集
	commandIndex[cmd.Name()] = cmd
	if !cmd.HasSubCommands() {
		return
	}
	buildCommandSetIndex(cmd.Name(), cmd.Commands())
}

func buildCommandSetIndex(prefix string, commands []*cobra.Command) {
	for _, command := range commands {
		fullName := commandName(prefix, command.Name())
		if !command.HasSubCommands() {
			commandIndex[fullName] = command
		} else {
			buildCommandSetIndex(fullName, command.Commands())
		}
	}
}

func (f *Flags) startDaemon(cmd *cobra.Command) bool {
	if !f.App.Flags.EnableDaemon {
		return false
	}
	// 隐藏的内建命令不启动后台进程，避免像下面的情况
	// 1. config init 命令启动了错误的常驻进程
	// 2. exec 这类频繁被调的命令没必要重启常驻进程
	if skipDaemon(cmd) {
		return false
	}
	dp := f.daemonProcess()
	info, err := dp.Start()
	if err != nil {
		f.App.Logger.Debugf("start daemon process failed: %v", err)
		return false
	}
	if info != nil {
		f.App.Logger.Debugf("start daemon process succeeded: %#v", info)
		return daemon.IsDaemon()
	}
	return false
}

func (f *Flags) daemonProcess() *daemon.AsyncProcess {
	return &daemon.AsyncProcess{
		Name: daemonName,
		// 传递版本，使得在 tc 更新版本时，得以重载守护进程
		Version:     f.App.Version,
		WorkDir:     f.App.WorkspaceLayout.RootPath,
		Args:        []string{daemonCommand.Name()},
		Singleton:   true,
		ProcessFile: filepath.Join(f.App.WorkspaceLayout.DaemonPath, fmt.Sprintf("%s.pid", daemonName)),
		LogFile:     filepath.Join(f.App.WorkspaceLayout.DaemonPath, fmt.Sprintf("%s.log", daemonName)),
	}
}

func commandName(prefix, name string) string {
	return prefix + nameSplit + name
}

func (f *Flags) initBuiltinCmd(rc *cobra.Command) {
	rc.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return errs.NewProcessErrorWithCode(err, errs.CodeParamInvalid)
	})

	addToRootCmd(rc, GetExecCommand(f.App))
	if f.App.Flags.EnableConfig {
		setConfigFlags()
		addToRootCmd(rc, configCommand)
	}

	addToRootCmd(rc, initCommand)
	setUpdateFlags()
	addToRootCmd(rc, updateCommand)
	if f.App.Flags.EnableDaemon {
		addToRootCmd(rc, daemonCommand)
	}
	AddInstallCommand(rc, f.App)
	addToRootCmd(rc, getCompletingCommand(rc))
}

func (f *Flags) initDynamicCommands(rc, cmd *cobra.Command, err error) error {
	if !f.App.Flags.EnableDynamic {
		return nil
	}
	if err != nil {
		return f.registerDynamicCommands(rc)
	}
	if !cmd.HasParent() {
		return f.registerDynamicCommands(rc)
	}
	// 存在子命令则说明属于命令集，可以与动态命令进行混排
	if cmd.HasSubCommands() {
		return f.registerDynamicCommands(rc)
	}
	return nil
}

func skipDaemon(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.Hidden && cmd != daemonCommand
}
