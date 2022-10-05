// Package repo 提供仓库的管理能力
package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/util/git"
)

// Repo GIT仓库
type Repo struct {
	RepoStashDir string
	AutoUpdate   bool
}

// Engine 仓库标准方法
type Engine interface {
	// Dir 获取仓库在本地的映射文件夹
	Dir(repo string) string
	// Enable 拉取仓库到本地
	Enable(repo string) error
}

// Dir 获取仓库保存路径
func (r *Repo) Dir(repo string) string {
	httpRepo, _ := git.ToHTTP(repo, true)
	pathStr := strings.TrimPrefix(strings.TrimSuffix(httpRepo, ".git"), "https://")
	paths := strings.Split(pathStr, "/")
	return cfile.Resolve(r.RepoStashDir, filepath.Join(paths...), true)
}

// Enable 同步仓库，不存在时拉取，存在时同步到最新
func (r *Repo) Enable(repo string) error {
	repoDir := r.Dir(repo)
	repoInstance, err := git.Instance(repoDir)
	if err != nil {
		return err
	}
	// 不需要拉取
	if !r.AutoUpdate && repoInstance.IsRepository() {
		return nil
	}
	repoInstance.Auth(config.RepoAuthUser, config.RepoAuthPwd)
	// 已存在时同步
	if repoInstance.IsRepository() {
		err = repoInstance.PullForce(git.Origin, git.Master)
		if err == nil {
			return nil
		}
		// TODO(blurooochen): 这里移除目录，可能在并发的场景下会导致别的进程读取文件出问题
		// 尝试移除走克隆逻辑
		_ = os.RemoveAll(repoDir)
	}
	// 不存在时克隆
	err = repoInstance.Clone(repo)
	if err != nil {
		return fmt.Errorf("拉取仓库失败：%s", err)
	}
	return nil
}
