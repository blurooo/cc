// Package daemon 守护进程
// 提供跨平台的进程守护能力
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/blurooo/cc/cli"
	"github.com/blurooo/cc/pkg/exit"
	"github.com/shirou/gopsutil/process"

	"github.com/gofrs/flock"
)

const (
	daemonFlag  = "_T_DAEMON_FLAG"
	daemonValue = "true"
)

const (
	processFileExt = ".info"
	processLockExt = ".lock"
	logFileExt     = ".log"
)

const defaultFileMode = 0640

var timeout = time.Second * 10

// Daemon 守护进程
type Daemon interface {
	Start() error
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	// 组合进程入参
	AsyncProcess

	// ExecPath 执行路径
	ExecPath string
	// PID 进程ID
	PID int
}

// AsyncProcess 使用异步进程执行当前任务
type AsyncProcess struct {
	// Name 进程命名
	Name string
	// Version 进程版本，进程版本如果发生变更，将会移除旧版本的进程，并启用新版本的进程
	Version string
	// Args 命令参数，默认继承当前进程的参数
	Args []string
	// WorkDir 进程工作目录，默认为当前目录（建议指定目录，避免当前目录被移除）
	WorkDir string
	// Stdin 进程标准输入
	Stdin []byte
	// Singleton 进程单例模式，同一时刻将只有一个进程可以被执行
	Singleton bool
	// ProcessFile 指定进程信息写入文件，默认为缓存目录下的 t_daemon/${name}.info
	ProcessFile string
	// LogFile 指定日志文件，默认为缓存目录下的 t_daemon/${name}.log
	LogFile string

	exit exit.Exit
}

// Start 重载异步进程
func (p *AsyncProcess) Start() (*ProcessInfo, error) {
	p.handleParams()
	if isAsyncProcess() {
		return p.loadChildProcess()
	}
	// 主进程只需要启动异步进程即可，不引入任何重逻辑，所有重逻辑都由异步进程执行
	return p.startAsyncProcess()
}

// IsDaemon 是否处于守护进程状态
func IsDaemon() bool {
	return isAsyncProcess()
}

func (p *AsyncProcess) handleParams() {
	p.LogFile = p.getLogFile()
	p.ProcessFile = p.getProcessInfoFile()
	p.exit = exit.New(timeout)
}

func (p *AsyncProcess) startAsyncProcess() (*ProcessInfo, error) {
	execPath := getExecutablePath()
	var args []string
	if p.Args != nil {
		args = p.Args
	} else {
		// 默认继承参数
		args = os.Args[1:]
	}
	envs := append(os.Environ(), getAsyncProcessEnv())
	// 异步重载当前进程，使任务进入异步进程执行
	pid, err := cli.New().RunParamsAsync(context.Background(), cli.Params{
		Name:  execPath,
		Args:  args,
		Pwd:   p.WorkDir,
		Env:   envs,
		Stdin: p.Stdin,
	})
	if err != nil {
		return nil, err
	}
	// 调用方在获得进程信息的时候，应该直接返回
	info := &ProcessInfo{PID: pid, ExecPath: execPath, AsyncProcess: *p}
	return info, nil
}

// isRunning 判断进程是否运行中
// 由于进程ID可以被复用，所以进程ID处于运行态不代表任务在运行中
// 结合进程执行的路径，和进程ID，可以更准确地进行判断
func isRunning(execPath string, pid int) bool {
	ps, err := process.NewProcess(int32(pid))
	if err != nil {
		return false
	}
	name, err := ps.Name()
	if err != nil {
		return false
	}
	return strings.HasSuffix(execPath, name)
}

// loadChildProcess 子进程核心任务
// 1. 锁竞争，实现进程唯一
// 2. 日志重定向
// 3. 最后一次异步任务记录
func (p *AsyncProcess) loadChildProcess() (*ProcessInfo, error) {
	// TODO(blurooochen): 考虑关闭文件句柄
	logFile, err := os.OpenFile(p.getLogFile(), appendFlag(), defaultFileMode)
	if err != nil {
		return nil, err
	}
	// 退出时关闭日志文件句柄
	defer p.exit.Listen(tryCloseFileHandle(logFile))
	// 异步进程标准输出重定向到文件
	os.Stdout = logFile
	os.Stderr = logFile
	log.SetOutput(logFile)
	info := p.getProcessInfo()
	infoFile := p.getProcessInfoFile()
	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	log.Printf("load async process: %s", data)
	log.Println("trying to start process...")
	if !p.Singleton {
		return nil, os.WriteFile(infoFile, data, defaultFileMode)
	}
	// 启用独占模式时，通过文件锁实现
	return p.handleSingletonProcess()
}

func (p *AsyncProcess) handleSingletonProcess() (*ProcessInfo, error) {
	lockFile := p.getProcessLockFile()
	log.Printf("trying to lock %s...", lockFile)
	ok, err := tryLockFile(lockFile)
	if err != nil {
		log.Printf("lock error: %s", err)
		return nil, err
	}
	info := p.getProcessInfo()
	// 无法锁定，说明锁已经被抢占了
	if !ok {
		log.Println("the lock failed and was preempted")
		return p.handleLockFail()
	}
	p.exit.Listen(tryCleanLockFileHandle(lockFile))
	log.Println("lock success")
	infoFile := p.getProcessInfoFile()
	// 进程信息文件存在的话，进入更新进程的分支
	if _, err := os.Stat(infoFile); err == nil {
		return p.handleProcessUpdate(infoFile)
	}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	return nil, os.WriteFile(infoFile, data, defaultFileMode)
}

// handleLockFail 处理锁定失败场景
// 可能是被同版本的异步进程占有，这个时候不需要做什么，通过返回子进程信息，提示调用方中断执行即可
// 但也可能是被某个未知的进程占有，此时就属于异常分支了
// 暂时通过判断最可能锁定该文件的进程是否处于运行态作为判断依据
// TODO(blurooochen): 准确判断持有锁文件的进程，参考 lsof
func (p *AsyncProcess) handleLockFail() (*ProcessInfo, error) {
	defaultErr := fmt.Errorf("unknown process which locked the %s", p.getProcessLockFile())
	infoFile := p.getProcessInfoFile()
	if _, err := os.Stat(infoFile); err != nil {
		return nil, defaultErr
	}
	info, err := getProcessInfo(infoFile)
	if err != nil {
		return nil, defaultErr
	}
	if isRunning(info.ExecPath, info.PID) {
		log.Printf("the lock has been preempted by process %d and nothing will be done", info.PID)
		return info, nil
	}
	return nil, defaultErr
}

// handleProcessUpdate 处理进程更新逻辑，进程如果不是首次被创建，则需要面对较多的逻辑
// 例如：原进程还在执行中，是否需要重载新版本的进程
func (p *AsyncProcess) handleProcessUpdate(infoFile string) (*ProcessInfo, error) {
	srcInfo, err := getProcessInfo(infoFile)
	if err != nil {
		return nil, err
	}
	dstInfo := p.getProcessInfo()
	// 旧进程如果不在，则比对版本，版本不同的话需要更新旧进程
	if !isRunning(srcInfo.ExecPath, srcInfo.PID) {
		return p.writeProcessInfo(dstInfo)
	}
	log.Printf("%s`s process %d is running...", srcInfo.Name, srcInfo.PID)
	if willReloadProcess(*srcInfo, dstInfo) {
		err = killProcess(srcInfo.PID)
		log.Printf("kill %d result: %s", srcInfo.PID, err)
		return p.writeProcessInfo(dstInfo)
	}
	log.Printf("the process is not changed")
	return srcInfo, nil
}

func (p *AsyncProcess) writeProcessInfo(info ProcessInfo) (*ProcessInfo, error) {
	data, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	infoFile := p.getProcessInfoFile()
	err = os.WriteFile(infoFile, data, defaultFileMode)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func tryCleanLockFileHandle(lockFile string) exit.Handle {
	return func(ctx context.Context, signal os.Signal) {
		err := unlockFile(lockFile)
		log.Printf("unlock file: %s, err: %v", lockFile, err)
		err = os.Remove(lockFile)
		log.Printf("try remove lock file: %s, err: %v", lockFile, err)
	}
}

func tryCloseFileHandle(f *os.File) exit.Handle {
	return func(ctx context.Context, signal os.Signal) {
		log.Printf("closing file: %s...", f.Name())
		_ = f.Close()
	}
}

func getProcessInfo(infoFile string) (*ProcessInfo, error) {
	data, err := os.ReadFile(infoFile)
	if err != nil {
		return nil, err
	}
	info := &ProcessInfo{}
	err = json.Unmarshal(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func killProcess(pid int) error {
	ps, err := process.NewProcess(int32(pid))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return ps.TerminateWithContext(ctx)
}

func tryLockFile(lockFile string) (bool, error) {
	fl := flock.New(lockFile)
	// 进程生命周期不需要解锁，结束时自动释放
	return fl.TryLock()
}

func unlockFile(lockFile string) error {
	fl := flock.New(lockFile)
	return fl.Unlock()
}

// 只要任务版本发生变更，进程就需要被重载
// 异步常驻进程在版本升级之后，如果有功能变更、BUG修复等，常驻进程必须进行重载才能使新版本生效
func willReloadProcess(src ProcessInfo, dst ProcessInfo) bool {
	return src.Version != dst.Version
}

func (p *AsyncProcess) getProcessInfoFile() string {
	if p.ProcessFile != "" {
		return p.ProcessFile
	}
	return p.Name + processFileExt
}

// getProcessLockFile 获取进程锁文件
// 通过关联版本，减少锁被旧版本进程抢占的问题
func (p *AsyncProcess) getProcessLockFile() string {
	if p.Version != "" {
		return p.Name + "." + p.Version + processLockExt
	}
	return p.Name + processLockExt
}

func (p *AsyncProcess) getLogFile() string {
	if p.LogFile != "" {
		return p.LogFile
	}
	return p.Name + logFileExt
}

func (p *AsyncProcess) getProcessInfo() ProcessInfo {
	return ProcessInfo{PID: os.Getpid(), AsyncProcess: *p, ExecPath: getExecutablePath()}
}

func appendFlag() int {
	return os.O_APPEND | os.O_CREATE | os.O_WRONLY
}

// 尝试获取可执行路径
func getExecutablePath() string {
	path, err := os.Executable()
	if err == nil {
		return path
	}
	path = os.Args[0]
	if filepath.IsAbs(path) {
		return path
	}
	lookedPath, err := exec.LookPath(path)
	if err == nil {
		return lookedPath
	}
	return path
}

func isAsyncProcess() bool {
	return os.Getenv(daemonFlag) == daemonValue
}

func getAsyncProcessEnv() string {
	return fmt.Sprintf("%s=%s", daemonFlag, daemonValue)
}
