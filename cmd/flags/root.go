// Package flags 命令行参数相关处理逻辑
package flags

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/errs"
	"github.com/blurooo/cc/ioc"
	"github.com/blurooo/cc/util/log"
	"github.com/blurooo/cc/util/reporter"
	"tencent2/tools/dev_tools/t2cli/common/flags"
	"tencent2/tools/dev_tools/t2cli/daemon"
	"tencent2/tools/dev_tools/t2cli/report"
)

const nameSplit = "."

const daemonName = "tc.daemon"

var commandIndex = map[string]*cobra.Command{}

var rootCmd = &cobra.Command{
	Use:               config.AliasName,
	Short:             "Tencent2 Devops CLI",
	Long:              "提供大仓开发相关的 DevOps 能力",
	ValidArgsFunction: flags.EnableFlagsCompletion,
	Version:           fmt.Sprintf("%s (%s.%s)", config.Version, runtime.GOOS, runtime.GOARCH),
}

// Execute flag执行入口
func Execute() error {
	return execute()
}

func execute() error {
	entry := reporter.NewEntry(os.Args)
	var err error
	defer func() {
		addEntry(entry, err)
	}()
	err = config.Init()
	if err != nil {
		return err
	}
	cobra.OnInitialize(initialize)
	// 注册内建命令
	initBuiltinCmd()
	// 仅在没有命中内置子命令时，注册动态指令，提高内置指令的运行性能
	// 如果只命中根命令也会启用动态指令，便于打印动态指令的帮助信息
	cmd, _, err := rootCmd.Find(os.Args[1:])
	defer func() {
		entry.Command = getCommandName(cmd)
	}()
	// 每次进程启动都需要异步启动一次守护进程
	// 这样能确保守护进程在被各种情况结束掉之后（机器重启之类），也能通过使用 tc 重新唤起
	// 任何时候都只会保留一条守护进程
	// 将常驻进程放在此处是避免因程序版本问题导致动态命令注册失败，但却也没办法启动自动更新能力来解决问题
	ok := startDaemon(cmd)
	if ok {
		return nil
	}
	// 处理动态命令
	err = initDynamicCommands(cmd, err)
	if err != nil {
		return err
	}
	// 成功处理自动补全时，不需要继续往下走
	if ok = handleComplete(); ok {
		return nil
	}
	err = rootCmd.Execute()
	if err != nil {
		err = handleKnownError(err)
	}
	return err
}

func handleComplete() bool {
	completeCmd := &cobra.Command{
		Use:                   cobra.ShellCompRequestCmd,
		Aliases:               []string{cobra.ShellCompNoDescRequestCmd},
		DisableFlagsInUseLine: true,
		Hidden:                true,
		DisableFlagParsing:    true,
	}
	rootCmd.AddCommand(completeCmd)
	defer rootCmd.RemoveCommand(completeCmd)
	cmd, args, err := rootCmd.Find(os.Args[1:])
	if err != nil {
		return false
	}
	if cmd != completeCmd {
		return false
	}
	cmd, args, err = rootCmd.Find(args)
	if err != nil {
		return false
	}
	// 识别动态子命令
	if !isDynamicCommand(cmd) {
		return false
	}
	err = handleDynamicComplete(cmd, args)
	return err == nil
}

func handleDynamicComplete(cmd *cobra.Command, args []string) error {
	// 移除掉所有日志相关的干扰性输出
	ioc.Log.SetOutput(io.Discard)
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
	if isUnknownCommandError(err) {
		return nil
	}
	return err
}

func isUnknownCommandError(err error) bool {
	return strings.HasPrefix(err.Error(), "unknown command")
}

func addEntry(entry report.Entry, err error) {
	traceID := os.Getenv(config.TCLITraceID)
	if traceID != "" {
		entry.TraceID = traceID
	}
	entry.Version = config.Version
	entry.Scene = config.AppName
	entry.End(err)
	// 重置进程执行失败的上报逻辑，对于 tc 来说，统一上报为插件执行错误
	var eErr *exec.ExitError
	if ok := errors.As(err, &eErr); ok {
		entry.ExitCode = errs.CodePluginExecError
	}
	ioc.Reporter.Add(entry)
}

func addToRootCmd(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
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

func startDaemon(cmd *cobra.Command) bool {
	// 隐藏的内建命令不启动后台进程，避免像下面的情况
	// 1. config init 命令启动了错误的常驻进程
	// 2. exec 这类频繁被调的命令没必要重启常驻进程
	if skipDaemon(cmd) {
		return false
	}
	dp := daemonProcess()
	info, err := dp.Start()
	if err != nil {
		ioc.Log.Debugf("启动守护进程出错：%v", err)
		return false
	}
	if info != nil {
		ioc.Log.Debugf("启动守护进程成功：%#v", info)
		return daemon.IsDaemon()
	}
	return false
}

func daemonProcess() *daemon.AsyncProcess {
	return &daemon.AsyncProcess{
		Name: daemonName,
		// 传递版本，使得在 tc 更新版本时，得以重载守护进程
		Version:     config.Version,
		WorkDir:     config.AppConfDir,
		Args:        []string{daemonCommand.Name()},
		Singleton:   true,
		ProcessFile: filepath.Join(config.DaemonDir, fmt.Sprintf("%s.pid", daemonName)),
		LogFile:     filepath.Join(config.DaemonDir, fmt.Sprintf("%s.log", daemonName)),
	}
}

func commandName(prefix, name string) string {
	return prefix + nameSplit + name
}

func initBuiltinCmd() {
	rootCmd.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return errs.NewProcessErrorWithCode(err, errs.CodeParamInvalid)
	})
	withCommonParams(rootCmd)

	registerExecCmd()
	registerConfigCmd()
	registerDaemonCmd()
	registerInstallCmd()
	registerCompletionCmd()
}

func initDynamicCommands(cmd *cobra.Command, err error) error {
	if err != nil {
		return registerDynamicCommands()
	}
	if !cmd.HasParent() {
		return registerDynamicCommands()
	}
	// 存在子命令则说明属于命令集，可以与动态命令进行混排
	if cmd.HasSubCommands() {
		return registerDynamicCommands()
	}
	return nil
}

func skipDaemon(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.Hidden && cmd != daemonCommand
}

func getCommandName(cmd *cobra.Command) string {
	if cmd == nil {
		return config.AliasName
	}
	name := cmd.Name()
	cmd.VisitParents(func(command *cobra.Command) {
		if command == rootCmd {
			return
		}
		name = command.Name() + "." + name
	})
	return name
}

func initialize() {
	if config.Debug {
		log.SetDebug(ioc.Log)
	}
}

func withCommonParams(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&config.Debug, "debug", "x", false, "调试模式")
	// 绑定环境变量
	if os.Getenv("DEBUG") == config.True {
		config.Debug = true
	}
}
