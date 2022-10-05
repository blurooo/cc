// Package helper 类 man 指令，提供指令文件的帮助获取能力，例如 tc help xx 可以在命令行渲染其帮助文档
package helper

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/charmbracelet/glamour"

	"tencent2/tools/dev_tools/t2cli/utils/cli"
)

const less = "less"

func renderMarkdown(content []byte) error {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)

	content, err := r.RenderBytes(content)
	if err != nil {
		return err
	}
	return showContent(content)
}

func showContent(content []byte) error {
	if _, err := exec.LookPath(less); err != nil {
		fmt.Println(string(content))
		return nil
	}
	return cli.Local().RunParamsInherit(context.TODO(), cli.Params{
		Name:  less,
		Args:  []string{"-r"},
		Stdin: content,
	})
}
