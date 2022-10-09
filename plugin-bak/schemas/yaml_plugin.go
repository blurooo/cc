package schemas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blurooo/cc/cli"
	"github.com/blurooo/cc/resource"
	"gopkg.in/yaml.v3"
)

type PluginSchema struct {
	Name       string            `yaml:"name" json:"name"`
	Desc       string            `yaml:"desc" json:"desc"`
	Version    string            `yaml:"version" json:"version"`
	Dependency *DependencySchema `yaml:"dep" json:"dep"`
	Resource   *ResourceSchema   `yaml:"resource" json:"resource"`
	Enter      *EnterSchema      `yaml:"enter" json:"enter"`
}

func (p *PluginSchema) Unmarshal(data []byte, path string) error {
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
		return data, fmt.Errorf("plugin protocol %s is not supported", ext)
	}
}

func (p *PluginSchema) UnmarshalJson(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p *PluginSchema) UnmarshalYaml(data []byte) error {
	return yaml.Unmarshal(data, p)
}

type DependencySchema struct {
	Plugins []*DependentPluginSchema `yaml:"plugins" json:"plugins"`
}

type DependentPluginSchema struct {
	Name     string   `yaml:"name" json:"name"`
	File     string   `yaml:"file" json:"file"`
	RepoFile RepoFile `yaml:"repo_file" json:"repo_file"`
}

type RepoFile struct {
	URL  string `yaml:"url" json:"url"`
	Ref  string `yaml:"ref" json:"ref"`
	File string `yaml:"file" json:"file"`
}

type ResourceSchema struct {
	Repos    []ResourceRepoSchema    `yaml:"repos"`
	Mirrors  []ResourceMirrorSchema  `yaml:"mirrors"`
	Archives []ResourceArchiveSchema `yaml:"archives"`
}

type ResourceRepoSchema struct {
	URL  map[string]string `yaml:"url"`
	Ref  string            `yaml:"ref"`
	Path string            `yaml:"path"`
}

type ResourceArchiveSchema struct {
	URL             map[string]string `yaml:"url"`
	Path            string            `yaml:"path"`
	RetainTopFolder bool              `yaml:"retain_top_folder"`
}

type ResourceMirrorSchema struct {
	URL  map[string]string `yaml:"url"`
	Path string            `yaml:"path"`
}

type EnterSchema struct {
	Command map[string]string `yaml:"command" json:"command"`
}

func (y *DependentPluginSchema) GetName() string {
	if y.Name != "" {
		return y.Name
	}
	return getPluginName(y.Filepath())
}

func (y *DependentPluginSchema) Filepath() string {
	if y.File != "" {
		return y.File
	}
	return y.RepoFile.File
}

func (y *YamlDepPlugin) Load() ([]byte, error) {
	if y.Schema.File == "" &&
		(y.Schema.RepoFile.URL == "" || y.Schema.RepoFile.File == "") {
		return nil, errors.New("one of <plugin.file> and <plugin.url + plugin.path> must be set")
	}
	if y.Schema.File != "" {
		return y.loadFile(y.Schema.File)
	}
	// TODO: implement repo_file
	return nil, nil
}

func (y *YamlDepPlugin) loadFile(file string) ([]byte, error) {
	if err := checkRealFile(file); err != nil {
		return nil, err
	}
	// always in command source path
	return os.ReadFile(filepath.Join(y.Context.CommandSourcePath, file))
}

type YamlResource struct {
	Context PluginContext
	Schema  ResourceSchema
}

// UnmarshalYAML implement yaml.Unmarshaler
func (y *YamlResource) UnmarshalYAML(unmarshal func(interface{}) error) error {
	s := ResourceSchema{}
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	y.Schema = s
	return nil
}

func (y *ResourceRepoSchema) Load(ctx context.Context, pc PluginContext) error {
	return nil
}

func (y *ResourceMirrorSchema) Load(ctx context.Context, pc PluginContext) error {
	toPath := pc.ResourcePath
	if y.Path != "" {
		toPath = filepath.Join(pc.ResourcePath, y.Path)
	}
	url, err := SelectAndParseResource(pc, y.URL)
	if err != nil {
		return err
	}
	if _, err = resource.Download(ctx, url, toPath); err != nil {
		return fmt.Errorf("download %s failed, %w", y.URL, err)
	}
	return nil
}

func (y *ResourceArchiveSchema) Load(ctx context.Context, pc PluginContext) error {
	archiver := resource.Archiver{RetainTopFolder: y.RetainTopFolder}
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	url, err := SelectAndParseResource(pc, y.URL)
	if err != nil {
		return err
	}
	path, err := resource.Download(ctx, url, tmp)
	if err != nil {
		return fmt.Errorf("download %s failed, %w", y.URL, err)
	}
	toPath := pc.ResourcePath
	if y.Path != "" {
		toPath = filepath.Join(pc.ResourcePath, y.Path)
	}
	return archiver.UnArchiver(path, toPath)
}

type YamlEnter struct {
	Context PluginContext
	Schema  EnterSchema
}

// UnmarshalYAML implement yaml.Unmarshaler
func (y *YamlEnter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	s := EnterSchema{}
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	y.Schema = s
	return nil
}

func (y *YamlEnter) Exec(ctx context.Context, opts ExecOpts) error {
	command, err := SelectAndParseResource(y.Context, y.Schema.Command)
	if err != nil {
		return err
	}
	shell := command + " " + cli.QuoteCommands(opts.Args)
	exec := &cli.Command{}
	return exec.RunParamsInherit(ctx, cli.Params{
		Shell: shell,
		Args:  opts.Args,
		Env:   opts.Envs,
	})
}

func checkRealFile(file string) error {
	if filepath.IsAbs(file) {
		return fmt.Errorf("absolute path %s is not allowed", file)
	}
	if strings.HasPrefix(file, "..") {
		return fmt.Errorf("path %s outside the command source path is not allowed", file)
	}
	return nil
}
