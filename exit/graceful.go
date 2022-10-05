package exit

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const signalBaseCode = 128

var exitSignals = []os.Signal{
	syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT,
}

var gracefulExits []*graceful

type graceful struct {
	pe            ProcessExit
	once          sync.Once
	mutex         sync.Mutex
	timeout       time.Duration
	signalHandles map[os.Signal][]Handle
	exitHandles   []Handle
}

type defaultProcessExit struct{}

// New 创建退出实例
func New(timeout time.Duration) Exit {
	g := &graceful{
		timeout: timeout,
	}
	g.pe = &defaultProcessExit{}
	gracefulExits = append(gracefulExits, g)
	return g
}

// Gracefully 优雅退出
func Gracefully(code int) {
	for _, exit := range gracefulExits {
		if len(exit.exitHandles) == 0 {
			continue
		}
		_ = exit.processHandles(exit.exitHandles, nil)
	}
	os.Exit(code)
}

// HandleError 处理错误
func HandleError(err error) {
	if err == nil {
		Gracefully(errs.CodeSuccess)
	}
	log.Default.Errorf("%s [TraceID: %s]", err, data.TraceID())
	var eErr *exec.ExitError
	// 优先继承进程退出码
	if ok := errors.As(err, &eErr); ok {
		Gracefully(eErr.ExitCode())
	} else {
		Gracefully(errs.CodeUnknown)
	}
}

// Listen 注册优雅退出处理过程
func (g *graceful) Listen(handle Handle) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.registerHandle(nil, handle)
}

// ListenSignal 监听某个退出信号
func (g *graceful) ListenSignal(signal os.Signal, handle Handle) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.registerHandle(signal, handle)
}

// Exit 退出当前进程
func (e defaultProcessExit) Exit(code int) {
	os.Exit(code)
}

func (g *graceful) registerHandle(s os.Signal, handle Handle) {
	g.once.Do(func() {
		g.signalHandles = map[os.Signal][]Handle{}
		c := make(chan os.Signal, 1)
		signal.Notify(c, exitSignals...)
		go g.startListen(c)
	})
	if s == nil {
		g.exitHandles = append(g.exitHandles, handle)
	} else {
		g.signalHandles[s] = append(g.signalHandles[s], handle)
	}
}

func (g *graceful) processHandles(handles []Handle, signal os.Signal) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(len(handles))
	done := make(chan bool)
	for _, handle := range handles {
		go func(handle Handle) {
			defer wg.Done()
			handle(ctx, signal)
		}(handle)
	}
	go func() {
		wg.Wait()
		done <- true
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *graceful) startListen(signal chan os.Signal) {
	s := <-signal
	if handles, ok := g.signalHandles[s]; ok {
		// TODO(blurooochen): handle err
		_ = g.processHandles(handles, s)
	}
	_ = g.processHandles(g.exitHandles, s)
	g.exit(s)
}

func (g *graceful) exit(signal os.Signal) {
	s, ok := signal.(syscall.Signal)
	if ok {
		g.pe.Exit(signalBaseCode + int(s))
	} else {
		g.pe.Exit(0)
	}
}
