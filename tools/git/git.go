package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blurooo/cc/errs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const (
	httpReg = `^http(s)?://([a-zA-Z0-9._-]*?(:[a-zA-Z0-9._-]*)?@)?` +
		`[a-zA-Z0-9._-]+(/[a-zA-Z0-9._-]+)+.git$`
	sshReg           = `^git@.+:.+/.+\.git$`
	httpSecurePrefix = "https://"
	httpPrefix       = "http://"
	sshPrefix        = "git@"
)

// ErrNotFound 找不到预期资源
var ErrNotFound = errors.New("not found")

// Git 操作属性
type Git struct {
	Path string

	user       string
	password   string
	repository *git.Repository
}

// Instance 获取一个GIT操作实例
// path GIT操作的仓库路径
func Instance(path string) (*Git, error) {
	if path == "" {
		pwd, err := os.Getwd()
		if err == nil {
			path = pwd
		}
	}
	g := &Git{
		Path: path,
	}
	err := g.open()
	if err != nil {
		return nil, err
	}
	return g, nil
}

// Auth 内置认证
func (g *Git) Auth(user, pwd string) {
	g.user = user
	g.password = pwd
}

// GetUrl 获取URL
func (g *Git) GetUrl(domain, group, project string) string {
	return fmt.Sprintf("https://%s/%s/%s", domain,
		strings.Trim(group, "/"),
		strings.TrimPrefix(project, "/"))
}

// IsRepository 是否属于GIT仓库
func (g *Git) IsRepository() bool {
	return g.repository != nil
}

// ToSsh 将url转换为ssh形式
func ToSsh(url string) (string, error) {
	if IsSsh(url) {
		return url, nil
	}
	if !IsHttp(url) {
		return "", errors.New("无法识别的url")
	}
	url = strings.Replace(url, httpPrefix, "", 1)
	url = strings.Replace(url, httpSecurePrefix, "", 1)
	httpArr := strings.Split(url, "@")
	httpTrunk := httpArr[0]
	if len(httpArr) > 1 {
		httpTrunk = httpArr[1]
	}
	url = sshPrefix + httpTrunk
	url = strings.Replace(url, "/", ":", 1)
	return url, nil
}

// ToHttp 将url转换为http形式
func ToHttp(url string, secure bool) (string, error) {
	if IsHttp(url) {
		return httpToHttpUrl(url, secure), nil
	}
	toHTTPPrefix := httpPrefix
	if secure {
		toHTTPPrefix = httpSecurePrefix
	}
	if IsSsh(url) {
		url = strings.Replace(url, ":", "/", 1)
		url = strings.Replace(url, sshPrefix, toHTTPPrefix, 1)
		return url, nil
	}
	return "", errors.New("无法识别的url")
}

// 将任意http形式的url转换为http或https形式
func httpToHttpUrl(url string, secure bool) string {
	isSecure := strings.Contains(url, httpSecurePrefix)
	if secure == isSecure {
		return url
	}
	if secure {
		url = strings.Replace(url, httpPrefix, httpSecurePrefix, 1)
	} else {
		url = strings.Replace(url, httpSecurePrefix, httpPrefix, 1)
	}
	return url
}

// IsSsh 是否ssh地址
func IsSsh(url string) bool {
	match, err := regexp.MatchString(sshReg, url)
	return err == nil && match
}

// IsHttp 是否http地址
func IsHttp(url string) bool {
	match, err := regexp.MatchString(httpReg, url)
	return err == nil && match
}

// IsGitUrl 是否git仓库地址
func IsGitUrl(url string) bool {
	return IsHttp(url) || IsSsh(url)
}

// Clone 克隆项目
// 支持http和ssh自动切换
func (g *Git) Clone(url string) error {
	r, err := git.PlainClone(g.Path, false, &git.CloneOptions{
		URL: url,
		Auth: &http.BasicAuth{
			Username: g.user,
			Password: g.password,
		},
		InsecureSkipTLS: true,
	})
	if err != nil {
		return errs.NewProcessErrorWithCode(err, errs.CodeRuntimeConfigUnready)
	}
	g.repository = r
	return nil
}

// ShadowClone 浅克隆项目
func (g *Git) ShadowClone(url string, branch string) error {
	r, err := git.PlainClone(g.Path, false, &git.CloneOptions{
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		URL:           url,
		Auth: &http.BasicAuth{
			Username: g.user,
			Password: g.password,
		},
		SingleBranch: true,
		Depth:        1,
	})
	if err != nil {
		return err
	}
	g.repository = r
	return nil
}

// open 打开仓库
func (g *Git) open() error {
	if g.repository != nil {
		return nil
	}
	r, err := git.PlainOpenWithOptions(g.Path, &git.PlainOpenOptions{DetectDotGit: true})
	if err == nil {
		g.repository = r
		return nil
	}
	if err == git.ErrRepositoryNotExists {
		return nil
	}
	err = fmt.Errorf("failed to open repository: %s, %w", g.Path, err)
	return errs.NewProcessErrorWithCode(err, errs.CodeRuntimeConfigUnready)
}

// PullForce 强制拉取指定远程仓库的指定分支
func (g *Git) PullForce(remote, branch string) error {
	wt, err := g.repository.Worktree()
	if err != nil {
		return err
	}
	opts := &git.PullOptions{
		RemoteName: remote,
		Auth: &http.BasicAuth{
			Username: g.user,
			Password: g.password,
		},
		Force:           true,
		InsecureSkipTLS: true,
	}
	if branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
	}
	err = wt.PullContext(context.Background(), opts)
	if err == nil {
		return nil
	}
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return errs.NewProcessErrorWithCode(err, errs.CodeRuntimeConfigUnready)
}

// Diff 获取工作区的变更内容
func (g *Git) Diff(fromCommitID, toCommitID string) ([]diff.FilePatch, error) {
	fromCommit, err := g.repository.CommitObject(plumbing.NewHash(fromCommitID))
	if err != nil {
		return nil, err
	}
	toCommit, err := g.repository.CommitObject(plumbing.NewHash(toCommitID))
	if err != nil {
		return nil, err
	}
	patch, err := fromCommit.Patch(toCommit)
	if err != nil {
		return nil, err
	}
	return patch.FilePatches(), nil
}

// DiffHead 获取某个提交ID以来的变更
func (g *Git) DiffHead(fromCommitID string) ([]diff.FilePatch, error) {
	toCommitID, err := g.Head()
	if err != nil {
		return nil, nil
	}
	return g.Diff(fromCommitID, toCommitID)
}

// Head 获取当前提交ID
func (g *Git) Head() (string, error) {
	head, err := g.repository.Head()
	if err != nil {
		return "", nil
	}
	return head.String(), nil
}

func (g *Git) GetRemoteUrl(remote string) (string, error) {
	r, err := g.repository.Remote(remote)
	if err != nil {
		return "", err
	}
	if len(r.Config().URLs) > 0 {
		return r.Config().URLs[0], nil
	}
	return "", fmt.Errorf("not remote url")
}

func (g *Git) GetRemoteUrls() ([]string, error) {
	remotes, err := g.repository.Remotes()
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(remotes))
	for _, remote := range remotes {
		urls = append(urls, remote.Config().URLs...)
	}
	return urls, nil
}

// LastChange 获取某个文件最后一次变更的提交ID
func (g *Git) LastChange(path string) (string, error) {
	if filepath.IsAbs(path) {
		relPath, err := filepath.Rel(g.RootPath(), path)
		if err != nil {
			return "", fmt.Errorf("目标文件 [%s] 应位于所在仓库目录下 %s", path, g.RootPath())
		}
		path = relPath
	}
	path = filepath.ToSlash(path)
	commitIDs, err := g.repository.Log(&git.LogOptions{
		FileName: &path,
	})
	if err != nil {
		return "", fmt.Errorf("获取目标文件 [%s] 的提交日志失败：%w", path, err)
	}
	commitID, err := commitIDs.Next()
	if err != nil {
		return "", ErrNotFound
	}
	return commitID.Hash.String(), nil
}

// RootPath 获取顶级目录
func (g *Git) RootPath() string {
	wt, err := g.repository.Worktree()
	if err != nil {
		return g.Path
	}
	return wt.Filesystem.Root()
}
