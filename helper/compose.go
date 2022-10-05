package helper

import (
	"path/filepath"
	"strings"

	"tencent2/tools/dev_tools/t2cli/common/cfile"
)

var helpFileExtList = []string{".md", ".MD"}

// Help 执行帮助指令
func Help(path string) error {
	commandFile := helpFile(path)
	return fileHelp(commandFile)
}

func helpFile(file string) string {
	ext := filepath.Ext(file)
	f := strings.TrimSuffix(file, ext)
	for _, ext := range helpFileExtList {
		commandFile := f + ext
		if !cfile.Exist(commandFile) {
			continue
		}
		return commandFile
	}
	return ""
}
