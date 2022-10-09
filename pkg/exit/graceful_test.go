package exit

import (
	"context"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

func Test_graceful_Listen(t *testing.T) {
	type fields struct {
		pe            ProcessExit
		timeout       time.Duration
		signalHandles map[os.Signal][]Handle
	}
	type args struct {
		handle Handle
	}

	var wg sync.WaitGroup

	tests := []struct {
		name     string
		fields   fields
		args     args
		hitCount int
	}{
		{
			name: "监听并触发",
			fields: fields{
				pe: &processExitMock{
					code: 143,
					t:    t,
					wg:   &wg,
				},
				timeout:       time.Millisecond,
				signalHandles: nil,
			},
			args: args{func(ctx context.Context, signal os.Signal) {
				t.Log("hint")
				defer wg.Done()
				t.Log(signal)
			}},
			hitCount: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &graceful{
				pe:            tt.fields.pe,
				timeout:       tt.fields.timeout,
				signalHandles: tt.fields.signalHandles,
			}
			wg.Add(tt.hitCount)
			g.Listen(tt.args.handle)
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
			wg.Wait()
		})
	}
}

type processExitMock struct {
	t    *testing.T
	code int
	wg   *sync.WaitGroup
}

func (m *processExitMock) Exit(code int) {
	m.t.Log("hint")
	defer m.wg.Done()
	if m.code != code {
		m.t.Errorf("exit() = %v, want %v", code, m.code)
	}
}
