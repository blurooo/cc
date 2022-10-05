package helper

import (
	"testing"
)

func Test_renderMarkdown(t *testing.T) {
	type args struct {
		content []byte
	}
	con := []byte(`
			# Hello World
			This is a simple example of Markdown rendering with Glamour!	
			Bye!
			`)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				content: con,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := renderMarkdown(tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("renderMarkdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
