package tc

import (
	"fmt"
	"strings"
	"sync"

	"github.com/blurooo/cc/config"
	"github.com/blurooo/cc/ioc"
)

var (
	pConfig     *config.PersistentConfig
	pConfigOnce sync.Once
	loadErr     error
)

// ValidConfigDetail 可用配置详情
type ValidConfigDetail struct {
	Key     string
	Comment string
}

// LoadConfig 加载持久化配置
func LoadConfig() config.PersistentConfig {
	pConfigOnce.Do(func() {
		c, err := load()
		if err != nil {
			pConfig = defaultConfig()
			loadErr = err
		} else {
			pConfig = c
		}
	})
	if loadErr != nil {
		ioc.Log.Warnf("无法加载配置：%s，请检查文件 [%s] 内容", loadErr, config.AppConfigFile)
	}
	return *pConfig
}

// SetConfig 保存配置
func SetConfig(key, value string) error {
	cfg, err := ini.LooseLoad(config.AppConfigFile)
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
	return cfg.SaveToIndent(config.AppConfigFile, "\t")
}

// GetConfig 获取配置
func GetConfig(keyName string) (string, error) {
	cfg, err := ini.LooseLoad(config.AppConfigFile)
	if err != nil {
		return "", fmt.Errorf("加载程序配置文件失败：%w", err)
	}
	sectionName, keyName := getSectionAndKeys(keyName)
	section := cfg.Section(sectionName)
	if section == nil {
		return "", fmt.Errorf("配置段 %s 不存在", sectionName)
	}
	key := section.Key(keyName)
	if key == nil {
		return "", fmt.Errorf("配置 %s.%s 不存在", sectionName, keyName)
	}
	return key.Value(), nil
}

// ListConfig 获取配置列表
func ListConfig() ([]string, error) {
	cfg, err := ini.LooseLoad(config.AppConfigFile)
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
func ListValidConfig() ([]ValidConfigDetail, error) {
	cfg := ini.Empty()
	// 从支持的配置里获取
	err := cfg.ReflectFrom(&config.PersistentConfig{})
	if err != nil {
		return nil, err
	}
	var validConfigs []ValidConfigDetail
	sections := cfg.Sections()
	for _, section := range sections {
		keys := section.Keys()
		for _, key := range keys {
			validConfig := ValidConfigDetail{
				Key:     fmt.Sprintf("%s.%s", section.Name(), key.Name()),
				Comment: key.Comment,
			}
			validConfigs = append(validConfigs, validConfig)
		}
	}
	return validConfigs, nil
}

// load 加载持久化配置
func load() (*config.PersistentConfig, error) {
	cfg, err := ini.LooseLoad(config.AppConfigFile)
	if err != nil {
		return nil, err
	}
	pConfig := defaultConfig()
	err = cfg.MapTo(pConfig)
	if err != nil {
		return nil, err
	}
	return pConfig, nil
}

func defaultConfig() *config.PersistentConfig {
	return &config.PersistentConfig{
		Update: config.Update{
			Always: true,
		},
	}
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
