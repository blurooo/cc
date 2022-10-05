package command

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/blurooo/cc/plugin"
	"github.com/blurooo/cc/repo"
	"tencent2/tools/dev_tools/t2cli/utils/cli"
)

type mockRepo struct {
	repo.Repo
	dir      string
	hasError bool
}

func (r *mockRepo) Dir(repo string) string {
	return r.dir
}

// Enable 同步仓库，不存在时拉取，存在时同步到最新
func (r *mockRepo) Enable(_ string) error {
	if r.hasError {
		return errors.New("")
	}
	return nil
}

func Test_repoSearcher_List(t *testing.T) {
	type fields struct {
		repo    repo.Engine
		repoURL string
	}
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(dir)
	cmd := filepath.Join(dir, "cmd")
	code := filepath.Join(cmd, "code")
	hidden := filepath.Join(cmd, ".hidden")
	err = os.MkdirAll(code, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	err = os.MkdirAll(hidden, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	mrFile := filepath.Join(code, "mr.yml")
	err = ioutil.WriteFile(mrFile, nil, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	crFile := filepath.Join(code, "cr.yml")
	err = ioutil.WriteFile(crFile, []byte(`command:
  linux: echo hello`), os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	helloFile := filepath.Join(cmd, "hello.yml")
	err = ioutil.WriteFile(helloFile, []byte(`
jobs:
  a:
    name: hello`), os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	unknownFile := filepath.Join(cmd, "unknown.unknown")
	err = ioutil.WriteFile(unknownFile, nil, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	hiddenFile := filepath.Join(hidden, "cr.yml")
	err = ioutil.WriteFile(hiddenFile, nil, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	crPlugin, err := plugin.NewPlugin(cli.Local(), crFile)
	if err != nil {
		t.Error(err)
		return
	}
	helloPlugin, err := plugin.NewPlugin(cli.Local(), helloFile)
	if err != nil {
		t.Error(err)
		return
	}
	codeNode := Node{
		Name:    "code",
		Desc:    nodeSetDesc,
		Dir:     cmd,
		AbsPath: code,
		IsLeaf:  false,
	}
	crNode := Node{
		Parent:  &codeNode,
		Name:    "cr",
		Dir:     code,
		AbsPath: crFile,
		IsLeaf:  true,
		Plugin:  crPlugin,
	}
	codeNode.Children = []Node{crNode}
	tests := []struct {
		name    string
		fields  fields
		want    []Node
		wantErr bool
	}{
		{
			name: "正常搜集到指令",
			fields: fields{repo: &mockRepo{
				dir: dir,
			}},
			want: []Node{
				codeNode,
				{
					Name:    "hello",
					Dir:     cmd,
					AbsPath: helloFile,
					IsLeaf:  true,
					Plugin:  helloPlugin,
				},
			},
		}, {
			name: "获取指令失败",
			fields: fields{repo: &mockRepo{
				dir:      dir,
				hasError: true,
			}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &repoSearcher{
				Repo:       tt.fields.repo,
				RepoURL:    tt.fields.repoURL,
				CommandDir: "cmd",
			}
			got, err := r.List()
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("List() got = %v, want %v", got, tt.want)
			}
		})
	}
}
