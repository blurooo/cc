package plugin_bak

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/resource"
	"github.com/blurooo/cc/util/cruntime"
)

// TCLISchema 插件描述协议
type TCLISchema struct {
	Name      string              `yaml:"-"`
	Desc      string              `yaml:"desc"`
	Dep       Dep                 `yaml:"dep"`
	Version   string              `yaml:"version"`
	Resources resource.Downloads  `yaml:"resource"`
	PreRun    map[string]Commands `yaml:"pre-run"`
	PostRun   map[string]Commands `yaml:"post-run"`
	PreLoad   map[string]Commands `yaml:"pre-load"`
	PostLoad  map[string]Commands `yaml:"post-load"`
	Command   map[string]Commands `yaml:"command"`
	Runtime   bool                `yaml:"runtime"`
}

// Dep 插件依赖协议
type Dep struct {
	Os     []string `yaml:"os"`
	Plugin []string `yaml:"plugin"`
}

// LoadDoneInfo 插件加载完成记录信息
type LoadDoneInfo struct {
	SchemaVersion string
	Version       string
	LoadTime      time.Time
}

// Commands 指令集
type Commands []string

// UnmarshalYAML 字符串或字符串数组转指令集
func (commands *Commands) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make([]string, 0)
	err := unmarshal(&v)
	if err == nil {
		*commands = v
		return nil
	}
	vStr := ""
	err = unmarshal(&vStr)
	if err == nil {
		*commands = []string{vStr}
	}
	return nil
}

// ResolveTCli 从文件解析出TCLI协议
func ResolveTCli(path string) (*TCLISchema, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plugin := &TCLISchema{}
	err = yaml.Unmarshal(data, plugin)
	if err != nil {
		return nil, fmt.Errorf("解析协议文件 [%s] 失败：%w", path, err)
	}
	// 未指定版本时，统一指定版本
	if plugin.Version == "" {
		plugin.Version = config.DefaultVersion
	} else {
		plugin.Version = validVersion(plugin.Version)
	}
	plugin.Name = pathToCommandName(path)
	return plugin, err
}

// Commands 获取执行入口
func (p *TCLISchema) Commands(workspace string) Commands {
	commands := selectCommands(p.Command)
	return p.handleCommands(workspace, commands)
}

// SelectCommands 选择指令集
func (p *TCLISchema) SelectCommands(workspace string, commands map[string]Commands) Commands {
	useCommands := selectCommands(commands)
	return p.handleCommands(workspace, useCommands)
}

func (p *TCLISchema) handleCommands(workspace string, commands Commands) Commands {
	validCommands := make(Commands, 0, len(commands))
	for _, command := range commands {
		command = strings.ReplaceAll(command, "${version}", p.Version)
		command = strings.ReplaceAll(command, "${ver}", p.Version)
		command = strings.ReplaceAll(command, "${os}", cruntime.GOOS())
		command = strings.ReplaceAll(command, "${arch}", cruntime.GOARCH())
		command = strings.ReplaceAll(command, "${var:pd}", workspace)
		validCommands = append(validCommands, command)
	}
	return validCommands
}

func validVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	return version
}

// selectCommands 按照当前操作系统选择指令
func selectCommands(commands map[string]Commands) Commands {
	os := cruntime.GOOS()
	osArch := os + "." + cruntime.GOARCH()
	if c, ok := commands[osArch]; ok {
		return c
	}
	if c, ok := commands[os]; ok {
		return c
	}
	if c, ok := commands["default"]; ok {
		return c
	}
	return nil
}

// pathToCommandName 提取文件最后一段作为命令名
func pathToCommandName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(path)
	return strings.TrimSuffix(base, ext)
}
