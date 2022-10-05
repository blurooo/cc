package tc

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
	"tencent2/tools/dev_tools/t2cli/utils/cli"

	"github.com/blurooo/cc/command"
	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/ioc"
	"github.com/blurooo/cc/linker"
	"github.com/blurooo/cc/mirror"
	"github.com/blurooo/cc/plugin"
	"github.com/blurooo/cc/resource"
)

const path = "cli-market/tc"

const maxPathRenameNumber = 100

// UpdateStrategy 程序更新策略
type UpdateStrategy struct {
	// 更新全部
	All bool
}

// Init 初始化程序
func Init(execPath string) (string, error) {
	sha256, err := cfile.SHA256(execPath)
	if err != nil {
		return "", fmt.Errorf("获取程序的SHA256值失败，请排查：%w", err)
	}
	resourceDir := config.ResourceDir
	curVersionPath := filepath.Join(resourceDir, sha256)
	toPath := filepath.Join(curVersionPath, filepath.Base(execPath))
	if execPath == toPath {
		return "", nil
	}
	if !cfile.DirExist(curVersionPath) {
		err = os.MkdirAll(curVersionPath, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("创建目录 %s 失败，请排查：%w", curVersionPath, err)
		}
	}
	validPath, err := handleToPath(toPath)
	if err != nil {
		return "", fmt.Errorf("移除无效文件 %s 失败，请排查：%w", toPath, err)
	}
	toPath = validPath
	err = cfile.Copy(execPath, toPath)
	if err != nil {
		return "", fmt.Errorf("复制程序文件 %s 到 %s 失败，请排查：%w", execPath, toPath, err)
	}
	err = cfile.GrantExecute(toPath)
	if err != nil {
		return "", fmt.Errorf("赋予程序 %s 执行权限失败，请排查：%w", toPath, err)
	}
	linkPath, err := linker.New(config.AliasName, config.BinDir, toPath, linker.OverrideAlways)
	if err != nil {
		return "", fmt.Errorf("将 %s 连接到 %s 失败，请排查：%w", toPath, linkPath, err)
	}
	// 获取除新版本外的其它目录，等待移除
	clearInvalidVersion(resourceDir, sha256)
	ioc.Log.Info("构建程序运行环境成功")
	ioc.Log.Infof("已从 %s 连接到 %s", linkPath, toPath)
	ioc.Log.Info("正在拉取最新命令集...")
	nodes, err := updateCommandSet()
	if err != nil {
		ioc.Log.Warnf("命令集拉取失败，请排查：%s", err)
		return toPath, nil
	}
	err = updateNodes(nodes, UpdateStrategy{All: false})
	if err != nil {
		ioc.Log.Warnf("子命令更新失败，请排查：%s", err)
		return toPath, nil
	}
	ioc.Log.Infof("程序已准备就绪，工作目录：%s（移除工作目录即完成清理）", config.AppConfDir)
	return toPath, nil
}

func handleToPath(toPath string) (string, error) {
	var err error
	for i := 0; i < maxPathRenameNumber; i++ {
		if !cfile.Exist(toPath) {
			return toPath, nil
		}
		err = os.Remove(toPath)
		if err == nil {
			return toPath, nil
		}
		dir := filepath.Dir(toPath)
		base := filepath.Base(toPath)
		// 尝试重命名
		toPath = filepath.Join(dir, fmt.Sprintf("_%d_%s", i, base))
	}
	return "", err
}

// UpdateTools 更新所有工具版本
func UpdateTools(strategy UpdateStrategy) error {
	newPath, err := updateSelf()
	if err != nil {
		return err
	}
	// 版本发生更新了，后续的更新逻辑走新版本
	// 传递不再自我更新的标志
	if newPath != "" {
		return reload(newPath)
	}
	nodes, err := updateCommandSet()
	if err != nil {
		return err
	}
	text := "已使用过的"
	if strategy.All {
		text = "全部"
	}
	ioc.Log.Infof("正在更新%s子命令...", text)
	err = updateNodes(nodes, strategy)
	if err != nil {
		return err
	}
	ioc.Log.Info("已完成更新，请开始使用新版本！！")
	return nil
}

func clearInvalidVersion(resourceDir, newVersionName string) {
	// 获取除新版本外的其它目录，等待移除
	clearPaths := readClearPaths(resourceDir, newVersionName)
	for _, clearPath := range clearPaths {
		_ = os.RemoveAll(clearPath)
	}
}

func readClearPaths(resourceDir string, skipPath string) []string {
	infos, err := ioutil.ReadDir(resourceDir)
	if err != nil {
		return nil
	}
	var clearPaths []string
	for _, info := range infos {
		if info.Name() == skipPath {
			continue
		}
		clearPaths = append(clearPaths, filepath.Join(resourceDir, info.Name()))
	}
	return clearPaths
}

func updateNodes(nodes []command.Node, strategy UpdateStrategy) error {
	for _, node := range nodes {
		err := updateNode(node, strategy)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateNode(node command.Node, strategy UpdateStrategy) error {
	if !node.IsLeaf {
		if len(node.Children) == 0 {
			return nil
		}
		return updateNodes(node.Children, strategy)
	}
	err := node.Plugin.Update(plugin.UpdateOpts{Lazy: !strategy.All})
	if err != nil {
		return fmt.Errorf("更新子命令 %s 失败：%w", node.AbsPath, err)
	}
	return nil
}

// updateSelf 程序自我更新，成功时，返回新版本路径，如果没有发生实际更新则新版本路径为空
func updateSelf() (string, error) {
	// 不需要更新的话跳过
	if hadUpdateSelf() {
		return "", nil
	}
	ioc.Log.Info("正在更新主框架...")
	ioc.Log.Info("正在请求制品库...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	info, err := mirror.Latest(ctx, path)
	if err != nil {
		return "", err
	}
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取程序所在地址失败：%w", err)
	}
	sha256, err := cfile.SHA256(exePath)
	if err != nil {
		return "", fmt.Errorf("计算程序的SHA25值失败：%w", err)
	}
	if sha256 == info.Sha256 {
		ioc.Log.Debug("已在使用最新版本，无须更新！！")
		return "", nil
	}
	ioc.Log.Infof("发现新版本，该版本创建于 %s，大小为：%s，SHA256值为：%s",
		info.CreatedDate, info.Size, info.Sha256)
	ioc.Log.Infof("下载地址：%s", info.URL)
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", fmt.Errorf("申请临时目录失败：%w", err)
	}
	defer os.RemoveAll(dir)
	toPath, err := resource.Download(info.URL, dir)
	if err != nil {
		return "", fmt.Errorf("资源下载失败：%w", err)
	}
	sha256, err = cfile.SHA256(toPath)
	if err != nil {
		return "", fmt.Errorf("计算文件的SHA25值失败：%w", err)
	}
	if sha256 != info.Sha256 {
		return "", fmt.Errorf("SHA256校验失败，下载到的文件SHA256为：%s，期望的SHA256为：%s", sha256, info.Sha256)
	}
	return Init(toPath)
}

// updateCommandSet 更新子命令集
func updateCommandSet() ([]command.Node, error) {
	ioc.Log.Info("正在更新子命令集...")
	searcher := getSearcher(true, config.CommandDir)
	nodes, err := searcher.List()
	if err != nil {
		return nil, fmt.Errorf("获取子命令集失败，请排查：%w", err)
	}
	return nodes, nil
}

func reload(execPath string) error {
	useEnvs := append(config.Envs, fmt.Sprintf("%s=%s", config.EnvUpdateSelf, "false"))
	return cli.Local().RunParamsInherit(context.TODO(), cli.Params{
		Name: execPath,
		Args: os.Args[1:],
		Env:  useEnvs,
	})
}

func hadUpdateSelf() bool {
	return os.Getenv(config.EnvUpdateSelf) == "false"
}
