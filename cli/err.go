package cli

import (
	"errors"
	"strconv"
)

// ErrExecTimeout is the error resulting if process timeout
var ErrExecTimeout = &Error{
	Name: "timeout",
	Err:  errors.New("child_process: Handler timeout"),
}

// ErrNotFound is the error resulting if a path search failed to find an executable file.
var ErrNotFound = errors.New("executable file not found in $PATH")

// Error 统一的执行错误结构
type Error struct {
	Name string
	Err  error
}

// Error 输出错误
func (e *Error) Error() string {
	return "exec: " + strconv.Quote(e.Name) + ": " + e.Err.Error()
}
