package helper

import (
	"fmt"
	"io/ioutil"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
	"github.com/blurooo/cc/ioc"
)

func fileHelp(filePath string) error {
	// 判断文件是否存在
	if !cfile.Exist(filePath) {
		return fmt.Errorf("期望的帮助文件不存在：%s", filePath)
	}
	ioc.Log.Infof("帮助文件路径: %s", filePath)
	// 读取文件内容
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return renderMarkdown(content)
}
