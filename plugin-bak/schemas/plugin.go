package schemas

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/blurooo/cc/cli"
	"github.com/blurooo/cc/pkg/linker"
	"github.com/blurooo/cc/tools/git"
)

type Plugin interface {
	PluginContext() PluginContext
	Execute(ctx context.Context, opts ExecOpts) error
	Load(ctx context.Context, opts LoadOpts) error
	Update(ctx context.Context, opts UpdateOpts) error
}

// SinglePlugin is a plugin implementer based on YAML/JSON syntax.
type SinglePlugin struct {
	Context  PluginContext
	Schema   *PluginSchema
	LoadInfo *LoadInfo
	Loader   *PluginLoader
}

type PluginContext struct {
	Path              string
	Sum               string
	Workspace         string
	CommandSourcePath string
	BinPath           string
	ResourcePath      string
	LoadFile          string
}

type LoadInfo struct {
	Sum      string
	LoadTime time.Time
}

func (p *SinglePlugin) Name() string {
	if p.Schema.Name != "" {
		return p.Schema.Name
	}
	return getPluginName(p.Context.Path)
}

func (p *SinglePlugin) Desc() string {
	return p.Schema.Desc
}

func (p *SinglePlugin) Version() string {
	if p.Schema.Version != "" {
		return p.Schema.Version
	}
	return "latest"
}

func (p *SinglePlugin) PluginContext() PluginContext {
	return p.Context
}

func (p *SinglePlugin) Execute(ctx context.Context, opts ExecOpts) error {
	command, err := SelectAndParseResource(p.Context, p.Schema.Enter.Command)
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

func (p *SinglePlugin) Load(ctx context.Context, opts LoadOpts) error {
	// If the plugin has already been loaded, ignore it
	if p.LoadInfo != nil && p.LoadInfo.Sum == p.Context.Sum {
		return nil
	}
	if err := p.loadDependencies(ctx, opts); err != nil {
		return err
	}
	if err := p.loadResources(ctx, opts); err != nil {
		return err
	}
	return p.writeLoadInfo()
}

func (p *SinglePlugin) Update(ctx context.Context, opts UpdateOpts) error {
	return nil
}

func (p *SinglePlugin) writeLoadInfo() error {
	info := &LoadInfo{
		Sum:      p.Context.Sum,
		LoadTime: time.Now(),
	}
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal load info failed, %w", err)
	}
	if err := os.WriteFile(p.Context.LoadFile, data, 0666); err != nil {
		return fmt.Errorf("write load info to %s failed, %w", p.Context.LoadFile, err)
	}
	return nil
}

func (p *SinglePlugin) loadDependencies(ctx context.Context, opts LoadOpts) error {
	if p.Schema.Dependency == nil {
		return nil
	}
	for _, plugin := range p.Schema.Dependency.Plugins {
		if err := p.loadDependentPlugin(ctx, plugin, opts); err != nil {
			return fmt.Errorf("load dependenct plugin %s failed, %w", plugin.GetName(), err)
		}
	}
	return nil
}

func (p *SinglePlugin) loadDependentPlugin(ctx context.Context, schema *DependentPluginSchema, opts LoadOpts) error {
	if err := p.checkDependentPlugin(schema); err != nil {
		return err
	}
	plugin, err := p.resolveDependentPlugin(ctx, schema)
	if err != nil {
		return err
	}
	// If not in lazy loading mode, the dependency should also be loaded immediately.
	if !opts.Lazy {
		if err := plugin.Load(ctx, opts); err != nil {
			return err
		}
	}
	pc := plugin.PluginContext()
	// call _exec subcommand to run dependent plugin
	command := fmt.Sprintf(`%s _exec "%s"`, p.Loader.Name, pc.Path)
	_, err = linker.New(schema.GetName(), pc.BinPath, command, linker.OverrideAlways)
	return err
}

func (p *SinglePlugin) checkDependentPlugin(schema *DependentPluginSchema) error {
	if schema.File == "" &&
		(schema.RepoFile.URL == "" || schema.RepoFile.File == "") {
		return errors.New("one of <plugin.file> and <plugin.url + plugin.path> must be set")
	}
	return checkRealFile(schema.Filepath())
}

func (p *SinglePlugin) resolveDependentPlugin(ctx context.Context, schema *DependentPluginSchema) (Plugin, error) {
	if schema.File != "" {
		// the dependent plugin file always in command source path
		return p.Loader.ResolvePath(ctx, filepath.Join(p.Context.CommandSourcePath, schema.File))
	}
	return p.Loader.ResolveRepoFile(ctx, schema.RepoFile.URL, schema.RepoFile.Ref, schema.RepoFile.File)
}

func (p *SinglePlugin) loadResources(ctx context.Context, opts LoadOpts) error {
	for _, mirror := range p.Schema.Resource.Mirrors {
		if err := mirror.Load(ctx, p.Context); err != nil {
			return fmt.Errorf("load mirror resource failed: %w", err)
		}
	}
	for _, archive := range p.Schema.Resource.Archives {
		if err := archive.Load(ctx, p.Context); err != nil {
			return fmt.Errorf("load archive resource failed: %w", err)
		}
	}
	for _, repo := range p.Schema.Resource.Repos {
		if err := repo.Load(ctx, p.Context); err != nil {
			return fmt.Errorf("load repository resource failed: %w", err)
		}
	}
	return nil
}

type ExecOpts struct {
	Envs []string
	Args []string
}

type LoadOpts struct {
	Update bool
	Lazy   bool
}

type UpdateOpts struct {
	Lazy bool
}

type PluginLoader struct {
	Name           string
	PluginRootPath string
	BinPath        string
}

func (l *PluginLoader) ResolvePath(_ context.Context, path string) (Plugin, error) {
	schema := &PluginSchema{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s failed, %w", path, err)
	}
	sum, err := getSum(data)
	if err != nil {
		return nil, fmt.Errorf("computing sum failed, data: %s, %w", data, err)
	}
	if err := schema.Unmarshal(data, path); err != nil {
		return nil, err
	}
	ins, err := git.Instance(path)
	if err != nil {
		return nil, err
	}
	url, err := ins.GetRemoteURL("origin")
	if err != nil {
		return nil, err
	}
	httpURL, _ := git.ToHTTP(url, true)
	paths := strings.TrimPrefix(strings.TrimSuffix(httpURL, ".git"), "https://")
	wd := filepath.Join(l.PluginRootPath, paths)
	pc := PluginContext{
		Path:              path,
		Sum:               sum,
		Workspace:         wd,
		CommandSourcePath: ins.RootPath(),
		BinPath:           filepath.Join(wd, ".bin"),
		ResourcePath:      filepath.Join(wd, ".resource"),
		LoadFile:          filepath.Join(wd, ".load_info"),
	}
	loadInfo, err := readLoadInfo(pc.LoadFile)
	if err != nil {
		return nil, err
	}
	return &SinglePlugin{
		Context:  pc,
		Schema:   schema,
		LoadInfo: loadInfo,
		Loader:   l,
	}, nil
}

func (l *PluginLoader) ResolveRepoFile(_ context.Context, repo, ref, path string) (Plugin, error) {
	return nil, errors.New("implement me")
}

func readLoadInfo(loadFile string) (*LoadInfo, error) {
	if _, err := os.Stat(loadFile); err != nil {
		return nil, nil
	}
	data, err := os.ReadFile(loadFile)
	if err != nil {
		return nil, err
	}
	info := &LoadInfo{}
	if err := json.Unmarshal(data, info); err != nil {
		return nil, fmt.Errorf("unmarshal load file %s failed, %w", loadFile, err)
	}
	return info, nil
}

// SelectAndParseResource will return the resource which matches current system
func SelectAndParseResource(context PluginContext, resource map[string]string) (string, error) {
	r, err := selectResource(resource)
	if err != nil {
		return "", err
	}
	t, err := template.New("").Parse(r)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err = t.Execute(&out, context); err != nil {
		return "", err
	}
	return out.String(), nil
}

func selectResource(resource map[string]string) (string, error) {
	oa := runtime.GOOS + "." + runtime.GOARCH
	if v, ok := resource[oa]; ok {
		return v, nil
	}
	if v, ok := resource[runtime.GOOS]; ok {
		return v, nil
	}
	if v, ok := resource["*"]; ok {
		return v, nil
	}
	return "", fmt.Errorf("not resource is valid for current system: %s", oa)
}

func getPluginName(path string) string {
	name := filepath.Base(path)
	return strings.TrimSuffix(name, "."+filepath.Ext(name))
}

func getSum(data []byte) (string, error) {
	m := sha256.New()
	if _, err := m.Write(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(m.Sum(nil)), nil
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
