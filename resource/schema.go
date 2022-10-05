package resource

import (
	"fmt"

	"github.com/blurooo/cc/util/cruntime"
)

// TDownload 下载资源
type TDownload struct {
	URL             ResStr `yaml:"url"`
	To              ResStr `yaml:"to"`
	UnArchiverTo    ResStr `yaml:"unarchiver-to"`
	RetainTopFolder bool   `yaml:"retain-top-folder"`
}

// ResStr 资源字符串
type ResStr string

// Downloads 链接集
type Downloads []TDownload

// UnmarshalYAML 字符串或字符串数组转指令集
func (downloads *Downloads) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make([]TDownload, 0)
	err := unmarshal(&v)
	if err == nil {
		*downloads = v
		return nil
	}
	d := TDownload{}
	err = unmarshal(&d)
	if err == nil {
		*downloads = []TDownload{d}
	}
	return nil
}

// UnmarshalYAML 字符串统一填充变量
func (s *ResStr) UnmarshalYAML(unmarshal func(interface{}) error) error {
	m := make(map[string]ResStr)
	err := unmarshal(&m)
	if err == nil {
		os := cruntime.GOOS()
		osArch := os + "." + cruntime.GOARCH()
		if v, ok := m[osArch]; ok {
			*s = v
		} else if v, ok := m[os]; ok {
			*s = v
		} else if v, ok := m["default"]; ok {
			*s = v
		} else {
			return fmt.Errorf("请提供当前操作系统对应的的url")
		}
		return nil
	}
	str := ""
	err = unmarshal(&str)
	if err != nil {
		return err
	}
	*s = ResStr(str)
	return nil
}
