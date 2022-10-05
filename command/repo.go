package command

import (
	"github.com/blurooo/cc/repo"
)

type repoSearcher struct {
	Repo       repo.Engine
	RepoURL    string
	CommandDir string
}

// RepoSearcher 仓库指令查找器
func RepoSearcher(repoURL string, repo repo.Engine, commandDir string) Searcher {
	return &repoSearcher{Repo: repo, RepoURL: repoURL, CommandDir: commandDir}
}

// List 从远程仓库搜集指令列表
func (r *repoSearcher) List() ([]Node, error) {
	searcher, err := r.toFileSearcher(r.RepoURL)
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
func (r *repoSearcher) toFileSearcher(repoURL string) (Searcher, error) {
	err := r.Repo.Enable(repoURL)
	if err != nil {
		return nil, err
	}
	return FileSearcher(r.Repo.Dir(repoURL), r.CommandDir), nil
}
