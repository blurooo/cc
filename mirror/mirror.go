// Package mirror 提供 tc 在制品库上的制品管理能力
package mirror

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blurooo/cc/ioc"
	"tencent2/tools/dev_tools/t2cli/utils/cli"
)

const latest = "latest"
const api = "https://mirrors.tencent.com/mirrors/api/generic"
const downloadURL = "https://mirrors.tencent.com/repository/generic/%s/%s"

// Info 制品信息
type Info struct {
	Name        string
	Sha256      string
	CreatedDate string
	URL         string
	Size        string
}

type resp struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type listResp struct {
	resp
	Data listData `json:"data"`
}

type listData struct {
	Records []record `json:"records"`
}

// record 记录
type record struct {
	Name        string `json:"name"`
	CreatedDate string `json:"createdDate"`
	Path        string `json:"path"`
	Sha256      string `json:"sha256"`
	FullPath    string `json:"fullPath"`
	Md5         string `json:"md5"`
	Size        string `json:"size"`
}

// Latest 获取最新版本
func Latest(ctx context.Context, path string) (*Info, error) {
	api := listAPI(path)
	list := &listResp{}
	rsp, err := resty.New().SetRetryCount(1).
		R().SetContext(ctx).SetResult(list).Get(api)
	if err != nil {
		ioc.Log.Warn("制品库接口请求失败，请尝试通过重装的方式升级版本")
		if rsp != nil {
			ioc.Log.Debugf("状态码：%d, 响应内容：%s", rsp.StatusCode(), rsp.Body())
		}
		return nil, err
	}
	record := findRecord(list.Data.Records)
	if record == nil {
		return nil, fmt.Errorf("找不到当前系统的记录，原始信息：%#v", list.Data.Records)
	}
	return recordToInfo(path, record), nil
}

func findRecord(records []record) *record {
	keywords := osKeywords()
	// Note: records 顺序是不可控的，应该以可控的 keywords 优先
	for _, keyword := range keywords {
		for _, record := range records {
			if strings.Contains(record.Name, keyword) {
				return &record
			}
		}
	}
	return nil
}

func getDownloadURL(path, name string) string {
	return fmt.Sprintf(downloadURL, getLatestPath(path), name)
}

func listAPI(path string) string {
	path = getLatestPath(path)
	return fmt.Sprintf("%s/list?full_path=%s", api, path)
}

func getLatestPath(path string) string {
	path = filepath.Join(path, latest)
	path = filepath.ToSlash(path)
	return path
}

func recordToInfo(path string, record *record) *Info {
	return &Info{
		Name:        record.Name,
		Sha256:      record.Sha256,
		CreatedDate: record.CreatedDate,
		URL:         getDownloadURL(path, record.Name),
		Size:        record.Size,
	}
}

func osKeywords() []string {
	os := getOS()
	// 查找优先级：
	// 1. 系统加架构
	// 2. 系统
	return []string{fmt.Sprintf("%s_%s", os, getArch()), os}
}

func getOS() string {
	// 编译出来的二进制包都是以 macos 命名的，所以这里需要转换一下
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func getArch() string {
	if runtime.GOOS != "darwin" {
		return runtime.GOARCH
	}
	data, _, err := cli.Local().RunShell(context.Background(), "uname -a")
	if err != nil {
		return runtime.GOARCH
	}
	// Note: 历史原因，tc 以前没有给 darwin arm64 用户专门编译 arm64 版本的二进制包，所以这部分用户从 runtime.GOARCH 中
	// 只能取到 amd64，也就永远无法升级到 arm64 版本，通过 uname -a 动态获取 macos 的信息，当出现 RELEASE_ARM64 字样时
	// 认定其为 arm64，待所有历史的 tc 都升级到 arm64 版本之后这段逻辑可以移除
	if bytes.Contains(data, []byte(`RELEASE_ARM64`)) {
		return "arm64"
	}
	return runtime.GOARCH
}
