//go:build !windows
// +build !windows

package cli

import (
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

var escapePattern = regexp.MustCompile(`[^\w@%+=:,./-]`)

// QuoteCommands converts an array parameter to a string
// example: []string{"a/*", "$a", "hello"} -> 'a/*' '$a' hello
func QuoteCommands(args []string) string {
	l := make([]string, len(args))

	for i, s := range args {
		l[i] = quote(s)
	}

	return strings.Join(l, " ")
}

func quote(s string) string {
	if len(s) == 0 {
		return "''"
	}
	if escapePattern.MatchString(s) {
		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}
	return s
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
	return cmd.Run()
}

func startCmd(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Allows child processes to be separated
	cmd.SysProcAttr.Setsid = true
	return cmd.Start()
}
