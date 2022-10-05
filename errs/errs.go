// Package errs 定义错误码
package errs

import (
	"errors"
	"fmt"
	"os/exec"
)

// ProcessError 进程错误
type ProcessError struct {
	code int
	Err  error
}

// Error 实现错误类型
func (p *ProcessError) Error() string {
	return fmt.Sprintf("[%d], %s", p.code, p.Err)
}

// Code 实现数据上报退出码获取
func (p *ProcessError) Code() int {
	return p.code
}

// ExitCode 进程退出码，优先继承进程的退出码
func (p *ProcessError) ExitCode() int {
	var eErr *exec.ExitError
	if ok := errors.As(p.Err, &eErr); ok {
		return eErr.ExitCode()
	}
	return p.code
}

// NewProcessErrorWithCode 新建进程退出错误
func NewProcessErrorWithCode(err error, code int) error {
	return &ProcessError{
		code: code,
		Err:  err,
	}
}
