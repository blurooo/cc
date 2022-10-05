// Package linker 提供命令在不同操作系统上的链接机制
package linker

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
	"tencent2/tools/dev_tools/t2cli/utils/cli"
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
	err := cfile.MkdirAll(binDir)
	if err != nil {
		return "", fmt.Errorf("创建目录 %s 失败：%s", binDir, err)
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
	// 兼容msys2终端
	return binPath, linkToMsys2(name, binDir, command, option)
}

func linkToWin32(name, binPath, command string, options Option) (string, error) {
	f := cmdFile(binPath, name)
	if !hasOption(options, OverrideAlways) && cfile.Exist(f) {
		return f, nil
	}
	return f, ioutil.WriteFile(f, cmdTemplate(command), fileMode)
}

// linkToMsys2 例如win10下的git bash
func linkToMsys2(name, binPath, command string, options Option) error {
	f := msys2File(binPath, name)
	if !hasOption(options, OverrideAlways) && cfile.Exist(f) {
		return nil
	}
	commands := cli.Parse(command)
	if len(commands) == 0 {
		return fmt.Errorf("指令解析为空: %s", command)
	}
	commandPath := commands[0]
	if cfile.Exist(commandPath) {
		unixCommandPath := cfile.ToUnixLikePath(commandPath)
		command = strings.Replace(command, commandPath, unixCommandPath, 1)
	}
	return ioutil.WriteFile(f, shellTemplate(command), fileMode)
}

func linkToUnixLike(name, binPath, command string, options Option) (string, error) {
	binFile := filepath.Join(binPath, name)
	if !hasOption(options, OverrideAlways) && cfile.Exist(binFile) {
		return binFile, nil
	}
	template := shellTemplate(command)
	err := ioutil.WriteFile(binFile, template, fileMode)
	if err != nil {
		return "", fmt.Errorf("命令连接失败: %s", err)
	}
	return binFile, nil
}

func cmdFile(binPath, name string) string {
	return filepath.Join(binPath, fmt.Sprintf(`%s.cmd`, name))
}

func cmdTemplate(command string) []byte {
	return []byte(fmt.Sprintf(`@echo off

%s %%*`, command))
}

func shellTemplate(command string) []byte {
	return []byte(fmt.Sprintf(`#!/bin/sh

%s "$@"`, command))
}

func msys2File(binPath, name string) string {
	return filepath.Join(binPath, name)
}

func hasOption(option Option, targetOption Option) bool {
	return option&targetOption == targetOption
}
