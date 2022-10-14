package config

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

type PersistentConfig struct {
	Update  Update  `ini:"update"`
	Command Command `ini:"command"`
	Repo    Repo    `ini:"repo"`
}

type Update struct {
	Always bool `ini:"always" comment:"always update the application automatically"`
}

type Command struct {
	Repo string `ini:"repo" comment:"config the command repository"`
	Path string `ini:"path" comment:"config the command path"`
}

type Repo struct {
	User     string `ini:"user" comment:"config the username to access the repository"`
	Password string `ini:"password" comment:"config the password to access the repository"`
}

// Configurator is a configurator to config the application with ini syntax.
type Configurator struct {
	ConfigFile string

	persistentConfig *PersistentConfig
}

// NewConfigurator create a new configurator to config the application.
func NewConfigurator(configFile string, defaultConfig PersistentConfig) (*Configurator, error) {
	cfg, err := loadConfigFile(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.MapTo(defaultConfig); err != nil {
		return nil, err
	}
	return &Configurator{
		ConfigFile:       configFile,
		persistentConfig: &defaultConfig,
	}, nil
}

// LoadConfig will load all configs as PersistentConfig
func (c *Configurator) LoadConfig() PersistentConfig {
	return *c.persistentConfig
}

// SetConfig set and save the config.
// If the value is null, the config item will be unsetted.
func (c *Configurator) SetConfig(key, value string) error {
	cfg, err := loadConfigFile(c.ConfigFile)
	if err != nil {
		return err
	}
	section, key := splitSectionAndKey(key)
	if section == "" {
		return fmt.Errorf("the section of %s is missing, expect: {section}.%s", key, key)
	}
	if value == "" {
		cfg.Section(section).DeleteKey(key)
	} else {
		cfg.Section(section).Key(key).SetValue(value)
	}
	return cfg.SaveToIndent(c.ConfigFile, "\t")
}

// GetConfig returns the value for the specified configuration.
// the name consisting of {section}.{key}.
func (c *Configurator) GetConfig(name string) (string, error) {
	cfg, err := loadConfigFile(c.ConfigFile)
	if err != nil {
		return "", err
	}
	sectionName, name := splitSectionAndKey(name)
	section := cfg.Section(sectionName)
	if section == nil {
		return "", fmt.Errorf("unkown section: %s", sectionName)
	}
	key := section.Key(name)
	if key == nil {
		return "", fmt.Errorf("unkown config item: %s.%s", sectionName, name)
	}
	return key.Value(), nil
}

// Item is a config item.
type Item struct {
	Key     string
	Value   string
	Comment string
}

// ListUsedConfigs will list configurations that have been set, excluding those that have not.
func (c *Configurator) ListUsedConfigs() ([]Item, error) {
	cfg, err := loadConfigFile(c.ConfigFile)
	if err != nil {
		return nil, err
	}
	return c.listConfigs(cfg)
}

// ListUsableConfigs will list all usable configurations, including those that have not been configured.
func (c *Configurator) ListUsableConfigs() ([]Item, error) {
	cfg := ini.Empty()
	if err := cfg.ReflectFrom(&PersistentConfig{}); err != nil {
		return nil, err
	}
	return c.listConfigs(cfg)
}

func (c *Configurator) listConfigs(cfg *ini.File) ([]Item, error) {
	var items []Item
	sections := cfg.Sections()
	for _, section := range sections {
		ss, err := c.listSectionConfigs(section)
		if err != nil {
			return nil, err
		}
		items = append(items, ss...)
	}
	return items, nil
}

func (c *Configurator) listSectionConfigs(section *ini.Section) ([]Item, error) {
	keys := section.Keys()
	items := make([]Item, 0, len(keys))
	for _, key := range keys {
		item := Item{
			Key:     fmt.Sprintf("%s.%s", section.Name(), key.Name()),
			Value:   key.Value(),
			Comment: key.Comment,
		}
		items = append(items, item)
	}
	return items, nil
}

func loadConfigFile(configFile string) (*ini.File, error) {
	cfg, err := ini.LooseLoad(configFile)
	if err != nil {
		return nil, fmt.Errorf("load config [%s] failed, %w", configFile, err)
	}
	return cfg, nil
}

func splitSectionAndKey(key string) (string, string) {
	keys := strings.Split(key, ".")
	section := ""
	if len(keys) > 1 {
		section = keys[0]
		key = strings.Join(keys[1:], ".")
	}
	return section, key
}
