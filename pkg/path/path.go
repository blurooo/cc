// Package path 提供环境变量PATH相关的操作函数
package path

import (
	"os"
	"strings"
)

const (
	path = "PATH"
)

func appendEnvPaths(prior bool, oPaths string, paths ...string) string {
	separator := string(os.PathListSeparator)
	envPaths := strings.Split(oPaths, separator)
	if prior {
		envPaths = append(paths, envPaths...)
	} else {
		envPaths = append(envPaths, paths...)
	}
	newPaths := distinct(envPaths)
	return strings.Join(newPaths, separator)
}

// UpdateEnvPaths 添加环境变量路径，仅当前进程有效
func UpdateEnvPaths(prior bool, paths ...string) error {
	return os.Setenv(path, GetEnvPaths(prior, paths...))
}

// GetEnvPaths 添加环境变量路径，并返回新的PATH字符串
func GetEnvPaths(prior bool, paths ...string) string {
	envPathStr := os.Getenv(path)
	newPathStr := appendEnvPaths(prior, envPathStr, paths...)
	return newPathStr
}

// distinct 数组去重
func distinct(list []string) []string {
	dList := make([]string, 0)
	// 空map不占内存空间
	dMap := map[string]struct{}{}
	for _, item := range list {
		_, ok := dMap[item]
		if ok {
			continue
		}
		dList = append(dList, item)
		dMap[item] = struct{}{}
	}
	return dList
}
