package helper

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/blurooo/cc/cli"
	"github.com/charmbracelet/glamour"
)

const lessCommand = "less"

// RenderContent will render the markdown content to stdout.
func RenderContent(content []byte) error {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)

	content, err := r.RenderBytes(content)
	if err != nil {
		return err
	}
	return showContent(content)
}

// RenderFile will render the markdown file to stdout.
func RenderFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file [%s] does not exist", path)
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return RenderContent(content)
}

// IsMarkdown will tell you whether the file is markdown or not.
func IsMarkdown(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".md"
}

func showContent(content []byte) error {
	if _, err := exec.LookPath(lessCommand); err == nil {
		return less(content)
	}
	_, err := os.Stdout.Write(content)
	return err
}

func less(content []byte) error {
	return cli.New().RunParamsInherit(context.Background(), cli.Params{
		Name:  lessCommand,
		Args:  []string{"-r"},
		Stdin: content,
	})
}
