package config

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

// PersistentConfig 持久化配置
type PersistentConfig struct {
	Update  Update  `ini:"update"`
	Command Command `ini:"command"`
}

// Update 更新策略配置
type Update struct {
	Always bool `ini:"always" comment:"自动进行版本更新"`
}

// Command 指令集配置
type Command struct {
	Repo string `ini:"repo" comment:"自定义指令仓库，例如 https://xx.git"`
	Path string `ini:"path" comment:"自定义指令目录，将动态指令集指向本地某个路径"`
}

type Configurator struct {
	ConfigFile       string
	PersistentConfig *PersistentConfig
}

func NewConfigurator(configFile string, defaultConfig PersistentConfig) (*Configurator, error) {
	cfg, err := ini.LooseLoad(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.MapTo(defaultConfig); err != nil {
		return nil, err
	}
	return &Configurator{
		ConfigFile:       configFile,
		PersistentConfig: &defaultConfig,
	}, nil
}

// UsageConfigDetail 可用配置详情
type UsageConfigDetail struct {
	Key     string
	Comment string
}

// LoadConfig 加载持久化配置
func (c *Configurator) LoadConfig() *PersistentConfig {
	return c.PersistentConfig
}

// SetConfig 保存配置
func (c *Configurator) SetConfig(key, value string) error {
	cfg, err := ini.LooseLoad(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("加载程序配置文件失败：%w", err)
	}
	section, key := getSectionAndKeys(key)
	if section == "" {
		return fmt.Errorf("请提供 %s 的 section 部分，例如 xx.%s", key, key)
	}
	if value == "" {
		cfg.Section(section).DeleteKey(key)
	} else {
		cfg.Section(section).Key(key).SetValue(value)
	}
	return cfg.SaveToIndent(c.ConfigFile, "\t")
}

// GetConfig 获取配置
func (c *Configurator) GetConfig(name string) (string, error) {
	cfg, err := ini.LooseLoad(c.ConfigFile)
	if err != nil {
		return "", fmt.Errorf("加载程序配置文件失败：%w", err)
	}
	sectionName, name := getSectionAndKeys(name)
	section := cfg.Section(sectionName)
	if section == nil {
		return "", fmt.Errorf("配置段 %s 不存在", sectionName)
	}
	key := section.Key(name)
	if key == nil {
		return "", fmt.Errorf("配置 %s.%s 不存在", sectionName, name)
	}
	return key.Value(), nil
}

// ListConfig 获取配置列表
func (c *Configurator) ListConfig() ([]string, error) {
	cfg, err := ini.LooseLoad(c.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("加载程序配置文件失败：%w", err)
	}
	var items []string
	sections := cfg.Sections()
	for _, section := range sections {
		keys := section.Keys()
		for _, key := range keys {
			items = append(items, fmt.Sprintf("%s.%s=%s", section.Name(), key.Name(), key.Value()))
		}
	}
	return items, nil
}

// ListValidConfig 获取支持的配置列表
func (c *Configurator) ListValidConfig() ([]UsageConfigDetail, error) {
	cfg := ini.Empty()
	// 从支持的配置里获取
	err := cfg.ReflectFrom(&PersistentConfig{})
	if err != nil {
		return nil, err
	}
	var validConfigs []UsageConfigDetail
	sections := cfg.Sections()
	for _, section := range sections {
		keys := section.Keys()
		for _, key := range keys {
			validConfig := UsageConfigDetail{
				Key:     fmt.Sprintf("%s.%s", section.Name(), key.Name()),
				Comment: key.Comment,
			}
			validConfigs = append(validConfigs, validConfig)
		}
	}
	return validConfigs, nil
}

func getSectionAndKeys(key string) (string, string) {
	keys := strings.Split(key, ".")
	section := ""
	if len(keys) > 1 {
		section = keys[0]
		key = strings.Join(keys[1:], ".")
	}
	return section, key
}
