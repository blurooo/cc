package command

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/blurooo/cc/plugin"
)

func Test_fileSearcher_List(t *testing.T) {
	type fields struct {
		Dir string
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
	pr := plugin.Resolver{}
	crPlugin, err := pr.ResolvePath(context.Background(), crFile)
	if err != nil {
		t.Error(err)
		return
	}
	helloPlugin, err := pr.ResolvePath(context.Background(), helloFile)
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
			name:   "正常搜集到指令",
			fields: fields{Dir: dir},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FileSearcher(tt.fields.Dir, "cmd")
			got, err := f.List()
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
