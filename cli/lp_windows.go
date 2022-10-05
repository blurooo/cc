package cli

import (
	"os"
	"path/filepath"
	"strings"
)

type empty struct{}

func chkStat(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if d.IsDir() {
		return os.ErrPermission
	}
	return nil
}

func hasExt(file string) bool {
	i := strings.LastIndex(file, ".")
	if i < 0 {
		return false
	}
	return strings.LastIndexAny(file, `:\/`) < i
}

func findExecutable(file string, exts []string) (string, error) {
	if len(exts) == 0 {
		return file, chkStat(file)
	}
	if hasExt(file) {
		if chkStat(file) == nil {
			return file, nil
		}
	}
	for _, e := range exts {
		if f := file + e; chkStat(f) == nil {
			return f, nil
		}
	}
	return "", os.ErrNotExist
}

// LookPaths 找到某个命令的所有可执行路径
// windows实现
func LookPaths(file string) ([]string, error) {
	var exts []string
	x := os.Getenv(`PATHEXT`)
	if x != "" {
		for _, e := range strings.Split(strings.ToLower(x), `;`) {
			if e == "" {
				continue
			}
			if e[0] != '.' {
				e = "." + e
			}
			exts = append(exts, e)
		}
	} else {
		exts = []string{".com", ".exe", ".bat", ".cmd"}
	}

	if strings.ContainsAny(file, `:\/`) {
		f, err := findExecutable(file, exts)
		if err == nil {
			return []string{f}, nil
		}
		return nil, &Error{file, err}
	}
	filter := map[string]empty{}
	executablePaths := make([]string, 0)
	if f, err := findExecutable(filepath.Join(".", file), exts); err == nil {
		filter[f] = empty{}
		executablePaths = append(executablePaths, f)
	}
	path := os.Getenv("path")
	for _, dir := range filepath.SplitList(path) {
		if f, err := findExecutable(filepath.Join(dir, file), exts); err == nil {
			if _, ok := filter[f]; ok {
				continue
			}
			executablePaths = append(executablePaths, f)
		}
	}
	if len(executablePaths) > 0 {
		return executablePaths, nil
	}
	return nil, &Error{file, ErrNotFound}
}
