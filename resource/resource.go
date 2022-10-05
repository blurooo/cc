// Package resource 资源下载、解压等操作
package resource

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
)

// Download 下载资源
// toPath 如果为文件夹，则下载到该文件夹
// 并尽可能地用资源服务器提供的文件命名（来自于response -> header -> Content-Disposition）
// 如果资源服务器没有提供，则文件会被命名为一串uuid
// 如果toPath为文件完整路径，则直接以toPath作为下载后的文件
func Download(ctx context.Context, url string, toPath string) (string, error) {
	filename := findFilename(url, toPath)
	err := download(ctx, url, filename)
	if err != nil {
		return "", fmt.Errorf("download %s failed, %w", url, err)
	}
	return filename, nil
}

func download(ctx context.Context, url, filename string) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	f, _ := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0755)
	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	_, err := io.Copy(io.MultiWriter(f, bar), resp.Body)
	return err
}

// 尝试获取文件名
// 如果toPath为指定文件，则使用toPath
// 否则尝试查找url
func findFilename(url string, toPath string) string {
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash == -1 {
		return createFileName()
	}
	filename := url[lastSlash+1:]
	firstQuestion := strings.Index(filename, "?")
	if firstQuestion != -1 {
		filename = filename[0:firstQuestion]
	}
	return filepath.Join(toPath, filename)
}

func createFileName() string {
	return uuid.New().String()
}
