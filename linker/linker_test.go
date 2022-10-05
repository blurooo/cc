package linker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	echo := "echo"
	tDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tDir)
	}()
	binPath := filepath.Join(tDir, "test", "bin")
	_, err = New(echo, binPath, echo)
	assert.NoError(t, err)
}
