//go:build windows
// +build windows

package cli

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// QuoteCommands converts an array parameter to a string
func QuoteCommands(args []string) string {
	for i, arg := range args {
		args[i] = syscall.EscapeArg(arg)
	}
	return strings.Join(args, " ")
}

func runInheritByCmd(cmd *exec.Cmd) error {
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = io.MultiWriter(cmd.Stdout, os.Stdout)
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, os.Stderr)
	}
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	return runCmd(cmd)
}

func runCmd(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
		CmdLine:    QuoteCommands(cmd.Args),
	}
	return cmd.Run()
}

func startCmd(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
		// Allows child processes to be separated
		CreationFlags: windows.DETACHED_PROCESS |
			windows.CREATE_NO_WINDOW |
			windows.CREATE_NEW_PROCESS_GROUP,
		CmdLine: QuoteCommands(cmd.Args),
	}
	return cmd.Start()
}
