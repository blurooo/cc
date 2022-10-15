package plugin

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
	"github.com/blurooo/cc/plugin/schema"
	"github.com/blurooo/cc/resource"
	"github.com/blurooo/cc/tools/git"
)

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

type Plugin interface {
	Context() Context
	Name() string
	Desc() string
	Version() string
	Execute(ctx context.Context, opts ExecOpts) error
	Load(ctx context.Context, opts LoadOpts) error
	Update(ctx context.Context, opts UpdateOpts) error
}

// MixedPlugin is a plugin implementer based on YAML/JSON/.. syntax.
type MixedPlugin struct {
	Pc       Context
	Schema   *schema.MixedPlugin
	LoadInfo *LoadInfo
	Loader   *Resolver
}

type Context struct {
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

func (p *MixedPlugin) Name() string {
	if p.Schema.Name != "" {
		return p.Schema.Name
	}
	return getPluginName(p.Pc.Path)
}

func (p *MixedPlugin) Desc() string {
	return p.Schema.Desc
}

func (p *MixedPlugin) Version() string {
	if p.Schema.Version != "" {
		return p.Schema.Version
	}
	return "latest"
}

func (p *MixedPlugin) Context() Context {
	return p.Pc
}

func (p *MixedPlugin) Execute(ctx context.Context, opts ExecOpts) error {
	if err := p.exec(ctx, p.Schema.PreRun, ExecOpts{Envs: opts.Envs}); err != nil {
		return fmt.Errorf("the plugin cannot be pre-run, %w", err)
	}
	if err := p.exec(ctx, p.Schema.Enter.Command, opts); err != nil {
		return err
	}
	if err := p.exec(ctx, p.Schema.PostRun, ExecOpts{Envs: opts.Envs}); err != nil {
		return fmt.Errorf("the plugin cannot be post-run, %w", err)
	}
	return nil
}

func (p *MixedPlugin) Load(ctx context.Context, opts LoadOpts) error {
	// If the plugin has already been loaded, ignore it
	if p.LoadInfo != nil && p.LoadInfo.Sum == p.Pc.Sum {
		return nil
	}
	if err := os.RemoveAll(p.Pc.Workspace); err != nil {
		return fmt.Errorf("remove plugin workspace [%s] failed, %w", p.Pc.Workspace, err)
	}
	if err := os.MkdirAll(p.Pc.Workspace, os.ModeDir); err != nil {
		return fmt.Errorf("create plugin workspace [%s] failed, %w", p.Pc.Workspace, err)
	}
	if err := p.exec(ctx, p.Schema.PreLoad, ExecOpts{}); err != nil {
		return fmt.Errorf("the plugin cannot be preloaded, %w", err)
	}
	if err := p.loadDependencies(ctx, opts); err != nil {
		return err
	}
	if err := p.loadResources(ctx, opts); err != nil {
		return err
	}
	if err := p.exec(ctx, p.Schema.PostLoad, ExecOpts{}); err != nil {
		return fmt.Errorf("the plugin cannot be postloaded, %w", err)
	}
	return p.writeLoadInfo()
}

func (p *MixedPlugin) Update(ctx context.Context, opts UpdateOpts) error {
	return nil
}

func (p *MixedPlugin) exec(ctx context.Context, command map[string]string, opts ExecOpts) error {
	cmd, err := selectAndParseResource(p.Pc, command)
	if err != nil {
		return err
	}
	if cmd == "" {
		return nil
	}
	if len(opts.Args) > 0 {
		cmd += " " + cli.QuoteCommands(opts.Args)
	}
	return cli.New().RunParamsInherit(ctx, cli.Params{
		Shell: cmd,
		Env:   opts.Envs,
	})
}

func (p *MixedPlugin) writeLoadInfo() error {
	info := &LoadInfo{
		Sum:      p.Pc.Sum,
		LoadTime: time.Now(),
	}
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal load info failed, %w", err)
	}
	if err := os.WriteFile(p.Pc.LoadFile, data, 0666); err != nil {
		return fmt.Errorf("write load info to %s failed, %w", p.Pc.LoadFile, err)
	}
	return nil
}

func (p *MixedPlugin) loadDependencies(ctx context.Context, opts LoadOpts) error {
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

func (p *MixedPlugin) loadDependentPlugin(ctx context.Context, dp *schema.DependentPlugin, opts LoadOpts) error {
	plugin, err := p.resolveDependentPlugin(ctx, dp)
	if err != nil {
		return err
	}
	// If not in lazy loading mode, the dependency should also be loaded immediately.
	if !opts.Lazy {
		if err := plugin.Load(ctx, opts); err != nil {
			return err
		}
	}
	pc := plugin.Context()
	// call _exec subcommand to run dependent plugin
	command := fmt.Sprintf(`%s __exec "%s"`, p.Loader.Name, pc.Path)
	_, err = linker.New(dp.GetName(), pc.BinPath, command, linker.OverrideAlways)
	return err
}

func (p *MixedPlugin) checkDependentPlugin(dp *schema.DependentPlugin) error {
	if dp.File == "" &&
		(dp.RepoFile == nil || dp.RepoFile.URL == "" || dp.RepoFile.File == "") {
		return errors.New("one of <plugin.file> and <plugin.url + plugin.path> must be set")
	}
	return checkRealFile(dp.Filepath())
}

func (p *MixedPlugin) resolveDependentPlugin(ctx context.Context, dp *schema.DependentPlugin) (Plugin, error) {
	if err := p.checkDependentPlugin(dp); err != nil {
		return nil, err
	}
	if dp.File != "" {
		// the dependent plugin file always in command source path
		return p.Loader.ResolvePath(ctx, filepath.Join(p.Pc.CommandSourcePath, dp.File))
	}
	return p.Loader.ResolveRepoFile(ctx, dp.RepoFile.URL, dp.RepoFile.Ref, dp.RepoFile.File)
}

func (p *MixedPlugin) loadResources(ctx context.Context, opts LoadOpts) error {
	for _, mirror := range p.Schema.Resource.Mirrors {
		if err := p.LoadResourceMirror(ctx, mirror); err != nil {
			return fmt.Errorf("load mirror resource failed: %w", err)
		}
	}
	for _, archive := range p.Schema.Resource.Archives {
		if err := p.loadResourceArchive(ctx, archive); err != nil {
			return fmt.Errorf("load archive resource failed: %w", err)
		}
	}
	for _, repo := range p.Schema.Resource.Repos {
		if err := p.loadResourceRepo(ctx, repo); err != nil {
			return fmt.Errorf("load repository resource failed: %w", err)
		}
	}
	return nil
}

func (p *MixedPlugin) loadResourceArchive(ctx context.Context, ra *schema.ResourceArchive) error {
	archiver := resource.Archiver{RetainTopFolder: ra.RetainTopFolder}
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	url, err := selectAndParseResource(p.Pc, ra.URL)
	if err != nil {
		return err
	}
	path, err := resource.Download(ctx, url, tmp)
	if err != nil {
		return fmt.Errorf("download %s failed, %w", ra.URL, err)
	}
	toPath := p.Pc.ResourcePath
	if ra.Path != "" {
		toPath = filepath.Join(p.Pc.ResourcePath, ra.Path)
	}
	return archiver.UnArchiver(path, toPath)
}

func (p *MixedPlugin) LoadResourceMirror(ctx context.Context, rm *schema.ResourceMirror) error {
	toPath := p.Pc.ResourcePath
	if rm.Path != "" {
		toPath = filepath.Join(p.Pc.ResourcePath, rm.Path)
	}
	url, err := selectAndParseResource(p.Pc, rm.URL)
	if err != nil {
		return err
	}
	if _, err = resource.Download(ctx, url, toPath); err != nil {
		return fmt.Errorf("download %s failed, %w", rm.URL, err)
	}
	return nil
}

func (p *MixedPlugin) loadResourceRepo(ctx context.Context, rr *schema.ResourceRepo) error {
	return errors.New("implement me")
}

type Resolver struct {
	Name           string
	PluginRootPath string
	BinPath        string
}

func (r *Resolver) ResolvePath(_ context.Context, path string) (Plugin, error) {
	s := &schema.MixedPlugin{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s failed, %w", path, err)
	}
	if err := s.Unmarshal(data, path); err != nil {
		return nil, err
	}
	gi, err := git.Instance(path)
	if err != nil {
		return nil, err
	}
	url, err := gi.GetRemoteUrl("origin")
	if err != nil {
		return nil, err
	}
	httpURL, _ := git.ToHttp(url, true)
	paths := strings.TrimPrefix(strings.TrimSuffix(httpURL, ".git"), "https://")
	wd := filepath.Join(r.PluginRootPath, paths)
	sum, err := getSum(data)
	if err != nil {
		return nil, fmt.Errorf("computing sum failed, data: %s, %w", data, err)
	}
	pc := Context{
		Path:              path,
		Sum:               sum,
		Workspace:         wd,
		CommandSourcePath: gi.RootPath(),
		BinPath:           filepath.Join(wd, ".bin"),
		ResourcePath:      filepath.Join(wd, ".resource"),
		LoadFile:          filepath.Join(wd, ".load_info"),
	}
	loadInfo, err := readLoadInfo(pc.LoadFile)
	if err != nil {
		return nil, err
	}
	return &MixedPlugin{
		Pc:       pc,
		Schema:   s,
		LoadInfo: loadInfo,
		Loader:   r,
	}, nil
}

func (r *Resolver) ResolveRepoFile(_ context.Context, repo, ref, path string) (Plugin, error) {
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

// selectAndParseResource will return the resource which matches current system
func selectAndParseResource(context Context, resource map[string]string) (string, error) {
	r, ok := selectResource(resource)
	if !ok {
		return "", nil
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

func selectResource(resource map[string]string) (string, bool) {
	oa := runtime.GOOS + "." + runtime.GOARCH
	if v, ok := resource[oa]; ok {
		return v, true
	}
	if v, ok := resource[runtime.GOOS]; ok {
		return v, true
	}
	if v, ok := resource["*"]; ok {
		return v, true
	}
	return "", false
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
