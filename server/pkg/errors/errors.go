// Package errors 提供项目统一的错误处理工具。
//
// 设计要点：
//   - Error 携带可读 Message + 业务错误码 Code。
//   - 通过 WithMessage / WithStack 包装底层错误而无需引入 fmt.Errorf。
//   - Exposable 标记错误是否可暴露给前端；只有暴露的错误才会通过协议 reason 透传。
//   - 通过 As/Is 兼容 Go 标准 errors 包。
package errors

import (
	stderrors "errors"
	"fmt"
	"runtime"
	"strings"
)

// Error 是项目统一的错误类型。
type Error struct {
	Code      int
	Message   string
	Exposable bool
	I18nKey   string
	I18nArgs  []any
	cause     error
	stack     []uintptr
}

// New 创建一个新的错误。format 支持 fmt.Sprintf 风格。
func New(code int, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		stack:   callers(2),
	}
}

// NewExposable 创建一个可对外暴露的错误。
func NewExposable(code int, format string, args ...any) *Error {
	e := New(code, format, args...)
	e.Exposable = true
	return e
}

// WithMessage 为已有 error 包裹上下文消息，返回新的 *Error。
// 如果原 err 已是 *Error，则保留其 Code / Exposable。
func WithMessage(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	var inner *Error
	if stderrors.As(err, &inner) {
		return &Error{
			Code:      inner.Code,
			Message:   msg + ": " + inner.Message,
			Exposable: inner.Exposable,
			I18nKey:   inner.I18nKey,
			I18nArgs:  inner.I18nArgs,
			cause:     err,
			stack:     callers(2),
		}
	}
	return &Error{
		Code:    0,
		Message: msg + ": " + err.Error(),
		cause:   err,
		stack:   callers(2),
	}
}

// WithStack 仅附加调用栈而不修改消息。
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	var inner *Error
	if stderrors.As(err, &inner) {
		if len(inner.stack) > 0 {
			return inner
		}
		inner.stack = callers(2)
		return inner
	}
	return &Error{
		Code:    0,
		Message: err.Error(),
		cause:   err,
		stack:   callers(2),
	}
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Message
}

// Unwrap 暴露底层错误以便 errors.Is / errors.As。
func (e *Error) Unwrap() error { return e.cause }

// Is 兼容标准库 errors.Is，按 Code 匹配。
func (e *Error) Is(target error) bool {
	var t *Error
	if !stderrors.As(target, &t) {
		return false
	}
	return e.Code != 0 && e.Code == t.Code
}

// Stack 返回调用栈文本（仅调试用）。
func (e *Error) Stack() string {
	if e == nil || len(e.stack) == 0 {
		return ""
	}
	frames := runtime.CallersFrames(e.stack)
	var sb strings.Builder
	for {
		frame, more := frames.Next()
		sb.WriteString(frame.Function)
		sb.WriteString("\n\t")
		sb.WriteString(frame.File)
		sb.WriteString(":")
		fmt.Fprintf(&sb, "%d", frame.Line)
		sb.WriteString("\n")
		if !more {
			break
		}
	}
	return sb.String()
}

// IsExposable 返回错误是否可对外暴露。
func IsExposable(err error) bool {
	var e *Error
	if stderrors.As(err, &e) {
		return e.Exposable
	}
	return false
}

// CodeOf 返回错误码（不存在则 0）。
func CodeOf(err error) int {
	var e *Error
	if stderrors.As(err, &e) {
		return e.Code
	}
	return 0
}

func callers(skip int) []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	return pcs[:n]
}
