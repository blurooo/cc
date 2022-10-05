package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// Command 命令
type Command struct{}

// New create a Executor instance
func New() Executor {
	return &Command{}
}

// Params 命令行执行通用参数
type Params struct {
	// Name 执行的程序名，如果有值，则 Shell 字段被忽视
	Name string
	// Args 执行的参数
	Args []string
	// Shell 执行的语句
	Shell string
	// Pwd 命令执行的目录
	Pwd string
	// Env 执行的环境变量
	Env []string
	// Stdin 简化标准输入的使用方式，如果不传递，则默认继承os.stdin
	Stdin []byte
	// Stdout 标准输出
	Stdout io.Writer
	// Stderr 标准错误输出
	Stderr io.Writer
}

// Executor 命令行执行接口，对应的实现可能有
// 1. 本地命令行执行
// 2. docker命令行执行
// 3. 远程服务器命令行执行
type Executor interface {
	// Run 简单使用命令名和参数列表执行，并获得标准输出和标准错误输出
	// 默认继承当前进程的环境变量、当前路径、标准输入等属性
	// 命令必须在环境变量PATH中可被寻找，无法执行 export/echo 等系统内建命令
	Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error)
	// RunInherit 同 Run，绑定当前进程的标准输入、标准输出、标准错误输出
	RunInherit(ctx context.Context, name string, args ...string) error
	// RunAsync 同 Run，异步执行，拥有独立的进程生命周期，不随着当前进程的结束而终止
	RunAsync(ctx context.Context, name string, args ...string) (int, error)
	// RunShell 使用字符串形式执行，基于 bash -c | cmd /C 等特性，支持较为复杂的语句
	RunShell(ctx context.Context, shell string) ([]byte, []byte, error)
	// RunShellInherit 同 RunShell，绑定当前进程的标准输入、标准输出、标准错误输出
	RunShellInherit(ctx context.Context, shell string) error
	// RunShellAsync 同 RunShell，异步执行，拥有独立的进程生命周期，不随着当前进程的结束而终止
	RunShellAsync(ctx context.Context, shell string) (int, error)
	// RunParams 使用自定义参数执行，允许定制较多参数，获得标准输出和标准错误输出
	RunParams(ctx context.Context, params Params) ([]byte, []byte, error)
	// RunParamsInherit 同 RunParams，绑定当前进程的标准输入、标准输出、标准错误输出
	RunParamsInherit(ctx context.Context, params Params) error
	// RunParamsAsync 同 RunParams，异步执行，拥有独立的进程生命周期，不随着当前进程的结束而终止
	RunParamsAsync(ctx context.Context, params Params) (int, error)
}

// Run 简单使用命令名和参数列表执行，并获得标准输出和标准错误输出
// 默认继承当前进程的环境变量、当前路径、标准输入等属性
// 命令必须在环境变量PATH中可被寻找，无法执行 export/echo 等系统内建命令
func (c *Command) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd, err := c.buildCmd(ctx, name, args...)
	if err != nil {
		return nil, nil, err
	}
	return output(cmd)
}

// RunInherit 同 Run，绑定当前进程的标准输入、标准输出、标准错误输出
func (c *Command) RunInherit(ctx context.Context, name string, args ...string) error {
	cmd, err := c.buildCmd(ctx, name, args...)
	if err != nil {
		return err
	}
	return runInheritByCmd(cmd)
}

// RunAsync 同 Run，异步执行，拥有独立的进程生命周期，不随着当前进程的结束而终止
func (c *Command) RunAsync(ctx context.Context, name string, args ...string) (int, error) {
	cmd, err := c.buildCmd(ctx, name, args...)
	if err != nil {
		return 0, err
	}
	return cmd.Process.Pid, startCmd(cmd)
}

// RunShell 使用字符串形式执行，基于 bash -c | cmd /C 等特性，支持较为复杂的语句
func (c *Command) RunShell(ctx context.Context, shell string) ([]byte, []byte, error) {
	terminal, args, err := selectShellAndArgs()
	if err != nil {
		return nil, nil, err
	}
	return c.RunParams(ctx, Params{
		Name: terminal,
		Args: append(args, shell),
	})
}

// RunShellInherit 同 RunShell，绑定当前进程的标准输入、标准输出、标准错误输出
func (c *Command) RunShellInherit(ctx context.Context, shell string) error {
	terminal, args, err := selectShellAndArgs()
	if err != nil {
		return err
	}
	return c.RunParamsInherit(ctx, Params{
		Name: terminal,
		Args: append(args, shell),
	})
}

// RunShellAsync 同 RunShell，异步执行，拥有独立的进程生命周期，不随着当前进程的结束而终止
func (c *Command) RunShellAsync(ctx context.Context, shell string) (int, error) {
	terminal, args, err := selectShellAndArgs()
	if err != nil {
		return 0, err
	}
	return c.RunParamsAsync(ctx, Params{
		Name: terminal,
		Args: append(args, shell),
	})
}

// RunParams 使用自定义参数执行，允许定制较多参数，获得标准输出和标准错误输出
func (c *Command) RunParams(ctx context.Context, params Params) ([]byte, []byte, error) {
	err := handleParams(&params)
	if err != nil {
		return nil, nil, err
	}
	cmd, err := c.buildCmdFromParam(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	return output(cmd)
}

// RunParamsInherit 同 RunParams，绑定当前进程的标准输入、标准输出、标准错误输出
func (c *Command) RunParamsInherit(ctx context.Context, params Params) error {
	err := handleParams(&params)
	if err != nil {
		return err
	}
	cmd, err := c.buildCmdFromParam(ctx, params)
	if err != nil {
		return err
	}
	return runInheritByCmd(cmd)
}

// RunParamsAsync 同 RunParams，异步执行，拥有独立的进程生命周期，不随着当前进程的结束而终止
func (c *Command) RunParamsAsync(ctx context.Context, params Params) (int, error) {
	err := handleParams(&params)
	if err != nil {
		return 0, err
	}
	cmd, err := c.buildCmdFromParam(ctx, params)
	if err != nil {
		return 0, err
	}
	return cmd.Process.Pid, startCmd(cmd)
}

func handleParams(params *Params) error {
	if params.Name != "" {
		return nil
	}
	terminal, args, err := selectShellAndArgs()
	if err != nil {
		return err
	}
	params.Name = terminal
	params.Args = append(args, params.Shell)
	return nil
}

// selectShellAndArgs 获取当前操作系统适用的终端类型及其字符串执行参数
func selectShellAndArgs() (shell string, shellArgs []string, err error) {
	if isWindows() {
		shell, err = exec.LookPath("cmd")
		if err != nil {
			return "", nil, err
		}
		return shell, []string{"/C"}, nil
	}
	shell, err = exec.LookPath("bash")
	if err != nil {
		// bash 不存在则降级到 sh
		shell, err = exec.LookPath("sh")
	}
	if err != nil {
		return "", nil, err
	}
	return shell, []string{"-c"}, nil
}

func (c *Command) buildCmdFromParam(ctx context.Context, params Params) (*exec.Cmd, error) {
	cmd, err := c.buildCmd(ctx, params.Name, params.Args...)
	if err != nil {
		return nil, err
	}
	if params.Stdin == nil {
		cmd.Stdin = os.Stdin
	} else if len(params.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(params.Stdin)
	}
	if params.Pwd != "" {
		cmd.Dir = params.Pwd
	}
	if len(params.Env) > 0 {
		cmd.Env = params.Env
	}
	if params.Stderr != nil {
		cmd.Stderr = params.Stderr
	}
	if params.Stdout != nil {
		cmd.Stdout = params.Stdout
	}
	return cmd, nil
}

func (c *Command) buildCmd(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, name, args...), nil
}

func output(cmd *exec.Cmd) ([]byte, []byte, error) {
	var stdout bytes.Buffer
	if cmd.Stdout == nil {
		cmd.Stdout = &stdout
	}
	var stderr bytes.Buffer
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}

	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	err := runCmd(cmd)
	return stdout.Bytes(), stderr.Bytes(), err
}

func timeLimit(f func() error, timeout time.Duration) error {
	if timeout <= 0 {
		return f()
	}
	errChan := make(chan error)
	go func() {
		e := f()
		errChan <- e
	}()
	select {
	case err := <-errChan:
		return err
	case <-time.After(timeout):
		return ErrExecTimeout
	}
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
