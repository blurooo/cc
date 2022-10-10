package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blurooo/cc/errs"
	"gopkg.in/yaml.v3"
)

type RichPlugin struct {
	Name       string            `yaml:"name" json:"name"`
	Desc       string            `yaml:"desc" json:"desc"`
	Version    string            `yaml:"version" json:"version"`
	Dependency *Dependency       `yaml:"dep" json:"dep"`
	Resource   *Resource         `yaml:"resource" json:"resource"`
	PreLoad    map[string]string `yaml:"pre_load"`
	PostLoad   map[string]string `yaml:"post_load"`
	PreRun     map[string]string `yaml:"pre_run"`
	PostRun    map[string]string `yaml:"post_run"`
	Enter      *Enter            `yaml:"enter" json:"enter"`
}

type Dependency struct {
	Plugins []*DependentPlugin `yaml:"plugins" json:"plugins"`
}

type DependentPlugin struct {
	Name     string    `yaml:"name" json:"name"`
	File     string    `yaml:"file" json:"file"`
	RepoFile *RepoFile `yaml:"repo_file" json:"repo_file"`
}

type RepoFile struct {
	URL  string `yaml:"url" json:"url"`
	Ref  string `yaml:"ref" json:"ref"`
	File string `yaml:"file" json:"file"`
}

type Resource struct {
	Repos    []*ResourceRepo    `yaml:"repos"`
	Mirrors  []*ResourceMirror  `yaml:"mirrors"`
	Archives []*ResourceArchive `yaml:"archives"`
}

type ResourceRepo struct {
	URL  map[string]string `yaml:"url"`
	Ref  string            `yaml:"ref"`
	Path string            `yaml:"path"`
}

type ResourceArchive struct {
	URL             map[string]string `yaml:"url"`
	Path            string            `yaml:"path"`
	RetainTopFolder bool              `yaml:"retain_top_folder"`
}

type ResourceMirror struct {
	URL  map[string]string `yaml:"url"`
	Path string            `yaml:"path"`
}

type Enter struct {
	Command map[string]string `yaml:"command" json:"command"`
}

func (y *DependentPlugin) GetName() string {
	if y.Name != "" {
		return y.Name
	}
	name := filepath.Base(y.Filepath())
	return strings.TrimSuffix(name, "."+filepath.Ext(name))
}

func (y *DependentPlugin) Filepath() string {
	if y.File != "" {
		return y.File
	}
	return y.RepoFile.File
}

func (p *RichPlugin) Unmarshal(data []byte, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s failed, %w", path, err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case "yaml", "yml":
		return p.UnmarshalYaml(data)
	case "json":
		return p.UnmarshalJson(data)
	default:
		return fmt.Errorf("plugin protocol %s is not supported, %w", ext, errs.ErrUnsupportedPlugin)
	}
}

func (p *RichPlugin) UnmarshalJson(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p *RichPlugin) UnmarshalYaml(data []byte) error {
	return yaml.Unmarshal(data, p)
}
