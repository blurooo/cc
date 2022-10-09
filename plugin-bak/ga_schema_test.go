package plugin_bak

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveWorkflow(t *testing.T) {
	type args struct {
		path string
	}
	data1 := `
on:
 push:
   branches:
     - develop
`
	data2 := `on: pre-commit`
	data3 := `on: [pre-commit,push]`

	data4 := `
on:
 file_change:
   files:
     - [.code.yml]
`

	dir, _ := ioutil.TempDir("", "test")
	file1 := filepath.Join(dir, "hello1.yml")
	file2 := filepath.Join(dir, "hello2.yml")
	file3 := filepath.Join(dir, "hello3.yml")
	file4 := filepath.Join(dir, "hello4.yml")
	defer os.RemoveAll(dir)
	_ = ioutil.WriteFile(file1, []byte(data1), os.ModePerm)
	_ = ioutil.WriteFile(file2, []byte(data2), os.ModePerm)
	_ = ioutil.WriteFile(file3, []byte(data3), os.ModePerm)
	_ = ioutil.WriteFile(file4, []byte(data4), os.ModePerm)
	condition1 := map[string]interface{}{"branches": []interface{}{"develop"}}
	condition2 := map[string]interface{}{"files": []interface{}{[]interface{}{".code.yml"}}}
	tests := []struct {
		name    string
		args    args
		want    *Workflow
		wantErr bool
	}{
		{
			"case1",
			args{file1},
			&Workflow{
				Interaction: Interaction{},
				On:          On{[]Listener{{"push", condition1}}},
			},
			false,
		},
		{
			"case2",
			args{file2},
			&Workflow{
				Interaction: Interaction{},
				On:          On{[]Listener{{Event: "pre-commit"}}},
			},
			false,
		},
		{
			"case3",
			args{file3},
			&Workflow{
				Interaction: Interaction{},
				On:          On{[]Listener{{Event: "pre-commit"}, {Event: "push"}}},
			},
			false,
		},
		{
			"case4",
			args{file4},
			&Workflow{
				Interaction: Interaction{},
				On:          On{[]Listener{{Event: "file_change", Conditions: condition2}}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveWorkflow(tt.args.path)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("ResolveWorkflow() = %v, want %v", got, tt.want)
			}
			if tt.wantErr != (err != nil) {
				t.Errorf("ResolveWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
