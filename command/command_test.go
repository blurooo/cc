package command

import (
	"testing"
)

func Test_isCommandFile(t *testing.T) {
	type args struct {
		extList []string
		file    string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"yaml", args{[]string{"yaml", "json"}, "xxx.yaml"}, true},
		{"yml", args{[]string{"yml", "json"}, "xxx.yml"}, true},
		{"json", args{[]string{"yml", "yaml"}, "xxx.json"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCommandFile(tt.args.extList, tt.args.file); got != tt.want {
				t.Errorf("isCommandFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_FullName(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		sep  string
		want string
	}{
		{
			name: "正常拼接，存在父节点",
			node: &Node{Parent: &Node{Name: "parent"}, Name: "child"},
			sep:  ".",
			want: "parent.child",
		},
		{
			name: "正常拼接，不存在父节点",
			node: &Node{Name: "child"},
			sep:  ".",
			want: "child",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.FullName(tt.sep); got != tt.want {
				t.Errorf("FullName() = %v, want %v", got, tt.want)
			}
		})
	}
}
