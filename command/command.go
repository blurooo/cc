// Package command 提供指令动态搜索能力
// 核心主要提供两种指令源，基于文件/基于仓库
// 核心能力是将文件树转换为指令树
package command

import (
	"github.com/blurooo/cc/plugin"
)

// Node 指令节点
type Node struct {
	// Parent 父亲节点
	Parent *Node
	// Name 指令名
	Name string
	// Desc 指令描述
	Desc string
	// Dir 指令存放的文件夹
	Dir string
	// RepoURL 指令提供的仓库
	RepoURL string
	// AbsPath 指令文件完整路径
	AbsPath string
	// Children 子节点列表
	Children []Node
	// Plugin 指令执行器
	Plugin plugin.Plugin
	// IsLeaf 是否叶子节点
	IsLeaf bool
}

// Searcher 指令查找抽象接口
type Searcher interface {
	// List 获取当前指令源的所有指令节点
	List() ([]Node, error)
}

// FullName 获取节点的完整名称
func (n *Node) FullName() string {
	var name string
	for cur := n; cur != nil; cur = cur.Parent {
		name = cur.Name + name
		if cur.Parent != nil {
			name = "." + name
		}
	}
	return name
}
