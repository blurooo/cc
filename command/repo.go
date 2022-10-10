package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blurooo/cc/tools/git"
)

type repoSearcher struct {
	AutoUpdate   bool
	AuthUser     string
	AuthPassword string
	RepoURL      string
	CommandDir   string
	RepoRootPath string
}

// RepoSearcher 仓库指令查找器
func RepoSearcher(repoURL string, commandDir string) Searcher {
	return &repoSearcher{RepoURL: repoURL, CommandDir: commandDir}
}

// List 从远程仓库搜集指令列表
func (r *repoSearcher) List() ([]Node, error) {
	searcher, err := r.toFileSearcher()
	if err != nil {
		return nil, err
	}
	nodes, err := searcher.List()
	if err != nil {
		return nil, err
	}
	r.fillNodes(nodes)
	return nodes, nil
}

// 填充指令，包含指令的仓库信息等
func (r *repoSearcher) fillNodes(nodes []Node) {
	for i := 0; i < len(nodes); i++ {
		// 仓库URL对具体的指令自然是明确的
		// 但如果指令属于文件夹
		if nodes[i].IsLeaf || len(nodes[i].Children) == 0 {
			nodes[i].RepoURL = r.RepoURL
		}
		r.fillNodes(nodes[i].Children)
	}
}

// 仓库搜索同样基于文件搜索，只是在文件搜索之前会同步拉取仓库
func (r *repoSearcher) toFileSearcher() (Searcher, error) {
	if err := r.pull(); err != nil {
		return nil, err
	}
	return FileSearcher(r.repoWorkspace(), r.CommandDir), nil
}

func (r *repoSearcher) repoWorkspace() string {
	httpRepo, _ := git.ToHTTP(r.RepoURL, true)
	pathStr := strings.TrimPrefix(strings.TrimSuffix(httpRepo, ".git"), "https://")
	paths := strings.Split(pathStr, "/")
	return filepath.Join(r.RepoRootPath, filepath.Join(paths...))
}

func (r *repoSearcher) pull() error {
	rw := r.repoWorkspace()
	if err := os.MkdirAll(rw, os.ModeDir); err != nil {
		return fmt.Errorf("create repository path [%s] failed, %w", rw, err)
	}
	gi, err := git.Instance(rw)
	if err != nil {
		return err
	}
	// if the repository is ready and does not need to be updated automatically, skip the pull
	if !r.AutoUpdate && gi.IsRepository() {
		return nil
	}
	gi.Auth(r.AuthUser, r.AuthPassword)
	if gi.IsRepository() {
		err = gi.PullForce("", "")
		if err == nil {
			return nil
		}
		_ = os.RemoveAll(rw)
	}
	err = gi.Clone(r.RepoURL)
	if err != nil {
		return fmt.Errorf("clone repository %s to %s failed, %w", r.RepoURL, rw, err)
	}
	return nil
}
