// Package config 维护程序的配置相关信息
// 1. 运行时相关配置（当前目录，项目根路径等）
// 2. 项目配置（当前项目对tcc的文件配置）
// 3. 应用配置（当前用户本地对tcc的全局文件配置）
// 4. flags 相关配置...
package config

import "fmt"

// Init 初始化配置
func Init() error {
	initIOC()
	initEnvs()
	err := initRuntime()
	if err != nil {
		return fmt.Errorf("初始化运行时配置失败：%w", err)
	}
	return nil
}
