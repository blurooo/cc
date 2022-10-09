package command

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/blurooo/cc/cli"
	"github.com/blurooo/cc/errs"
	"github.com/blurooo/cc/plugin"
	"gopkg.in/yaml.v3"
)

const nodeSetDesc = "指令集合，使用 -h 获取详细"

// fileSearcher 基于文件系统的指令查找器
type fileSearcher struct {
	// RootDir 查找的根路径
	RootDir string
	// CommandDir 被查找的文件夹
	CommandDir string
	Resolver   *plugin.Resolver

	cli cli.Executor
}

type info struct {
	Desc string `yaml:"desc"`
}

// FileSearcher 文件指令查找器
func FileSearcher(dir, commandDir string) Searcher {
	cmdDir := filepath.Join(dir, commandDir)
	return &fileSearcher{
		CommandDir: cmdDir,
		RootDir:    dir,

		cli: cli.New(),
	}
}

// List 从文件系统中搜集指令列表
func (f *fileSearcher) List() ([]Node, error) {
	return f.collectDirNodes(f.CommandDir, nil)
}

func (f *fileSearcher) collectDirNodes(dir string, parent *Node) ([]Node, error) {
	var nodes []Node
	wErr := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if path == dir {
				return nil
			}
			if err != nil {
				return errs.NewProcessErrorWithCode(err, errs.CodeFileOperationFail)
			}
			if invalidFilePath(info) {
				return walkError(info)
			}
			var node *Node
			if info.IsDir() {
				node, err = f.dirToNode(path, parent)
			} else {
				node, err = f.fileToNode(path, parent)
			}
			if err != nil {
				if err == plugin.ErrUnSupported {
					return nil
				}
				return err
			}
			if node == nil {
				return nil
			}
			node.AbsPath = path
			node.Dir = filepath.Dir(path)
			nodes = append(nodes, *node)
			return walkError(info)
		})
	return nodes, wErr
}

func invalidFilePath(info os.FileInfo) bool {
	return info == nil || strings.HasPrefix(info.Name(), ".")
}

func walkError(info os.FileInfo) error {
	if info.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

func (f *fileSearcher) dirToNode(dir string, parent *Node) (*Node, error) {
	node := &Node{
		Parent: parent,
		IsLeaf: false,
		Name:   filepath.Base(dir),
		Desc:   dirDesc(dir),
	}
	children, err := f.collectDirNodes(dir, node)
	if err != nil {
		return nil, err
	}
	node.Children = children
	return node, nil
}

func (f *fileSearcher) fileToNode(path string, parent *Node) (*Node, error) {
	p, err := f.Resolver.ResolvePath(context.Background(), path)
	if err != nil {
		return nil, err
	}
	return &Node{
		Parent:  parent,
		IsLeaf:  true,
		Name:    p.Name(),
		Dir:     filepath.Dir(path),
		Desc:    p.Desc(),
		Plugin:  p,
		AbsPath: path,
	}, nil
}

func dirDesc(dir string) string {
	path := filepath.Join(dir, ".info")
	if _, err := os.Stat(path); err != nil {
		return nodeSetDesc
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nodeSetDesc
	}
	info := &info{}
	err = yaml.Unmarshal(data, info)
	if err != nil {
		return nodeSetDesc
	}
	return info.Desc
}
