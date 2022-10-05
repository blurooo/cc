package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGit_Dir(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)
	type fields struct {
		RepoStashDir string
		AutoUpdate   bool
	}
	type args struct {
		repo string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "正常获取项目目录",
			fields: fields{
				RepoStashDir: dir,
			},
			args: args{repo: "http://git.oa.com/a/b/c.git"},
			want: filepath.Join(dir, "git.oa.com/a/b/c"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Repo{
				RepoStashDir: tt.fields.RepoStashDir,
				AutoUpdate:   tt.fields.AutoUpdate,
			}
			if got := r.Dir(tt.args.repo); got != tt.want {
				t.Errorf("CommandDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
