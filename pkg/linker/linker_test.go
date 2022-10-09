package linker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	echo := "echo"
	binPath := filepath.Join(t.TempDir(), "test", "bin")
	if _, err := New(echo, binPath, echo); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(binPath, echo)); err != nil {
		t.Error(err)
	}
}
