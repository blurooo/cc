//go:build !windows
// +build !windows

package cli

import (
	"os"
	"path/filepath"
	"strings"
)

type empty struct{}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}

// LookPaths 找到某个命令的所有可执行路径
// unix-like平台实现
// 这段代码来自于 exec.LookPath，但 exec.LookPath 在找到一个路径之后就返回了
// 此方法可以找到所有的执行路径
func LookPaths(file string) ([]string, error) {
	if strings.Contains(file, "/") {
		err := findExecutable(file)
		if err == nil {
			return []string{file}, nil
		}
		return nil, &Error{file, err}
	}
	path := os.Getenv("PATH")
	filter := map[string]empty{}
	executablePaths := make([]string, 0)
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(path); err == nil {
			if _, ok := filter[path]; ok {
				continue
			}
			executablePaths = append(executablePaths, path)
		}
	}
	if len(executablePaths) > 0 {
		return executablePaths, nil
	}
	return nil, ErrNotFound
}
