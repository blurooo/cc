// Package linker 提供命令在不同操作系统上的链接机制
package linker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/blurooo/cc/cli"
)

const fileMode = 0744

// Option 连接选项
type Option int

const (
	// None 什么操作都没有
	None Option = iota
	// OverrideAlways 总是覆盖
	OverrideAlways = 1 << 0
)

// New 注册指令
// 例如 New('test', '/usr/local/bin', 'echo')
// 会在 /usr/local/bin/test 下创建 test 并指向 echo "$@"，后续执行 test 即等于执行 echo
func New(name, binDir, command string, options ...Option) (string, error) {
	if err := os.MkdirAll(binDir, os.ModeDir); err != nil {
		return "", fmt.Errorf("create directory [%s] failed, %w", binDir, err)
	}
	option := None
	if len(options) > 0 {
		option = options[0]
	}
	if runtime.GOOS != "windows" {
		return linkToUnixLike(name, binDir, command, option)
	}
	binPath, err := linkToWin32(name, binDir, command, option)
	if err != nil {
		return "", err
	}
	// 兼容 msys2 终端
	return binPath, linkToMsys2(name, binDir, command, option)
}

func linkToWin32(name, binPath, command string, options Option) (string, error) {
	f := cmdFile(binPath, name)
	if !hasOption(options, OverrideAlways) && exist(f) {
		return f, nil
	}
	return f, ioutil.WriteFile(f, cmdTemplate(command), fileMode)
}

// linkToMsys2 例如 win10 下的 git bash
func linkToMsys2(name, binPath, command string, options Option) error {
	f := msys2File(binPath, name)
	if !hasOption(options, OverrideAlways) && exist(f) {
		return nil
	}
	commands := cli.Parse(command)
	if len(commands) == 0 {
		return errors.New("command is empty")
	}
	commandPath := commands[0]
	if exist(commandPath) {
		unixCommandPath := ToUnixLikePath(commandPath)
		command = strings.Replace(command, commandPath, unixCommandPath, 1)
	}
	return ioutil.WriteFile(f, shellTemplate(command), fileMode)
}

func linkToUnixLike(name, binPath, command string, options Option) (string, error) {
	binFile := filepath.Join(binPath, name)
	if !hasOption(options, OverrideAlways) && exist(binFile) {
		return binFile, nil
	}
	template := shellTemplate(command)
	err := ioutil.WriteFile(binFile, template, fileMode)
	if err != nil {
		return "", fmt.Errorf("link failed: %s", err)
	}
	return binFile, nil
}

func cmdFile(binPath, name string) string {
	return filepath.Join(binPath, fmt.Sprintf(`%s.cmd`, name))
}

func cmdTemplate(command string) []byte {
	return []byte(fmt.Sprintf("@echo off\n\n%s %%*", command))
}

func shellTemplate(command string) []byte {
	return []byte(fmt.Sprintf("#!/bin/sh\n\n%s \"$@\"", command))
}

func msys2File(binPath, name string) string {
	return filepath.Join(binPath, name)
}

func hasOption(option Option, targetOption Option) bool {
	return option&targetOption == targetOption
}

func exist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ToUnixLikePath 转换 windows 风格的路径为类 unix 风格
// C:\Users\Administrator\Go\src\blurooochen\git-hook\git-hook.exe
// ===>
// /c/Users/Administrator/Go/src/blurooochen/git-hook/git-hook.exe
func ToUnixLikePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if match, err := regexp.MatchString(`^[A-Za-z]:/.*`, path); err == nil && match {
		// 将 C:/Users/... 转换为 /c/Users/... 的形式
		path = "/" + strings.ToLower(path[0:1]) + path[2:]
	}
	return path
}
