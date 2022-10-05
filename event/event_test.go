package event

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestEmit_Pub(t *testing.T) {
	type fields struct {
		Subscribers map[string][]subscriber
	}
	type args struct {
		eventType string
		payload   map[string]interface{}
	}
	sub1 := subscriber{Conditions: map[string]interface{}{
		"branches": []interface{}{"dev"},
	}, Handler: func(conditions, payload map[string]interface{}) error {
		return nil
	},
	}
	sub2 := subscriber{Conditions: map[string]interface{}{
		"others": []interface{}{"xxx.csv"},
	}, Handler: func(conditions, payload map[string]interface{}) error {
		return nil
	},
	}
	sub3 := subscriber{Conditions: map[string]interface{}{
		"files": []interface{}{"xxx.json"},
	}, Handler: func(conditions, payload map[string]interface{}) error {
		return errors.New("error")
	},
	}
	fields1 := fields{
		Subscribers: map[string][]subscriber{"pre-commit": {sub1}, "files": {sub2, sub3}},
	}
	payload1 := map[string]interface{}{
		"branch": interface{}("dev"),
	}
	payload2 := map[string]interface{}{
		"files": []interface{}{"xxx.json"},
	}
	data := `
jobs:
  test:
    name: test
    steps:
      - name: echo
        id: shell
        run: |
          echo hello world
`
	dir, _ := ioutil.TempDir("", "test")
	file := filepath.Join(dir, "test.yml")

	defer os.RemoveAll(dir)
	_ = ioutil.WriteFile(file, []byte(data), os.ModePerm)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"pub success",
			fields1,
			args{"pre-commit", payload1},
			false,
		},
		{
			"not match event",
			fields1,
			args{"others", payload1},
			false,
		},
		{
			"pub fail",
			fields1,
			args{"files", payload2},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Emit{
				Subscribers: tt.fields.Subscribers,
			}
			if err := s.Pub(tt.args.eventType, tt.args.payload); (err != nil) != tt.wantErr {
				t.Errorf("Pub() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
