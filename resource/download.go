package resource

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
)

// Resource 资源实例
type Resource struct {
	Workspace string
	Version   string
}

// Download 下载资源
func (r *Resource) Download(resources Downloads) error {
	for _, resource := range resources {
		err := r.downloadResource(resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Resource) downloadResource(resource TDownload) error {
	url := strings.TrimSpace(string(resource.URL))
	if url == "" {
		return fmt.Errorf("url is empty")
	}
	url = r.fillInfo(url)
	to := string(resource.To)
	unArchiverTo := string(resource.UnArchiverTo)
	if to == "" && unArchiverTo == "" {
		to = r.Workspace
	}
	if to != "" {
		to = r.fillInfo(to)
		to = r.getPath(to)
		defer func() {
			_ = cfile.AmendFileOwner(to)
		}()
		toPath, err := Download(url, to)
		if err != nil {
			return err
		}
		return cfile.GrantExecute(toPath)
	}
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	toPath, err := Download(url, tmp)
	if err != nil {
		return fmt.Errorf("资源拉取失败，请检查网络并重试：%w", err)
	}
	if unArchiverTo != "" {
		unArchiverTo = r.fillInfo(unArchiverTo)
		unArchiverTo = r.getPath(unArchiverTo)
		defer func() {
			_ = cfile.AmendFileOwner(unArchiverTo)
		}()
		return UnArchiver(toPath, unArchiverTo, Params{RetainTopFolder: resource.RetainTopFolder})
	}
	return nil
}

func (r *Resource) getPath(path string) string {
	if !filepath.IsAbs(path) {
		return filepath.Join(r.Workspace, path)
	}
	return path
}

func (r *Resource) fillInfo(str string) string {
	template.New("").Parse(str)
	str = strings.ReplaceAll(str, "${ver}", r.Version)
	str = strings.ReplaceAll(str, "${os}", runtime.GOOS)
	str = strings.ReplaceAll(str, "${arch}", runtime.GOARCH)
	str = strings.ReplaceAll(str, "${pd}", r.Workspace)
	return str
}
