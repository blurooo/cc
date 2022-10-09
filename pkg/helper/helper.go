package helper

import (
	"fmt"
)

// Help will render the help info to screen.
func Help(path string) error {
	if IsMarkdown(path) {
		return RenderFile(path)
	}
	return fmt.Errorf("unsupported help file: %s", path)
}
