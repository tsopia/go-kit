package errors

import (
	"fmt"
	"net/http"
	"sync"
)

// Registry 保存错误定义注册信息。
type Registry struct {
	mu       sync.RWMutex
	byCode   map[int]*Definition
	byName   map[string]*Definition
	autoCode int
}

// Definition 描述一个错误定义。
type Definition struct {
	Code  int
	Name  string
	Class *Definition
	HTTP  int
}

// ErrorInfo 是导出的错误定义快照。
type ErrorInfo struct {
	Code  int
	Name  string
	Class string
}

// Error 让 Definition 可作为 errors.Is 的 target 使用。
func (d *Definition) Error() string {
	if d == nil {
		return "[]"
	}
	return "[" + d.Name + "]"
}

type codedError struct {
	def     *Definition
	message string
	cause   error
}

func (e *codedError) Error() string {
	name := ""
	if e != nil && e.def != nil {
		name = e.def.Name
	}
	if e != nil && e.message != "" {
		return "[" + name + "] " + e.message
	}
	return "[" + name + "]"
}

func (e *codedError) Is(target error) bool {
	if e == nil || e.def == nil || target == nil {
		return false
	}

	switch t := target.(type) {
	case *codedError:
		if t == nil || t.def == nil {
			return false
		}
		return e.def.Code == t.def.Code
	case *Definition:
		return e.matchesDefinition(t)
	default:
		return false
	}
}

func (e *codedError) matchesDefinition(target *Definition) bool {
	if target == nil || e == nil || e.def == nil {
		return false
	}

	for cur := e.def; cur != nil; cur = cur.Class {
		if cur.Code == target.Code {
			return true
		}
	}
	return false
}

func (e *codedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

const defaultAutoCodeStart = 4000 // 自动分配错误码起始值

var global = NewRegistry()

var (
	// Sentinel 错误定义，code 范围 2000-2999。
	NotFound     = Register(2001, "NOT_FOUND").WithHTTP(http.StatusNotFound)
	InvalidParam = Register(2002, "INVALID_PARAM").WithHTTP(http.StatusBadRequest)
	Unauthorized = Register(2003, "UNAUTHORIZED").WithHTTP(http.StatusUnauthorized)
	Forbidden    = Register(2004, "FORBIDDEN").WithHTTP(http.StatusForbidden)
	Internal     = Register(2005, "INTERNAL_ERROR").WithHTTP(http.StatusInternalServerError)
	Timeout      = Register(2006, "TIMEOUT").WithHTTP(http.StatusGatewayTimeout)
	BadGateway   = Register(2007, "BAD_GATEWAY").WithHTTP(http.StatusBadGateway)
)

// NewRegistry 创建注册表，自动编码从 4000 开始。
func NewRegistry() *Registry {
	return &Registry{
		byCode:   make(map[int]*Definition),
		byName:   make(map[string]*Definition),
		autoCode: defaultAutoCodeStart,
	}
}

// Register 使用指定 code/name 注册定义。
// 重复 code 或 name 都视为编程错误并 panic。
func (r *Registry) Register(code int, name string) *Definition {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byCode[code]; exists {
		panic("errors: duplicate code")
	}
	if _, exists := r.byName[name]; exists {
		panic("errors: duplicate name")
	}

	def := &Definition{Code: code, Name: name}
	r.byCode[code] = def
	r.byName[name] = def

	return def
}

// New 为 name 创建或返回已存在定义。
func (r *Registry) New(name string) *Definition {
	// 先读检查（无锁）
	r.mu.RLock()
	if def, exists := r.byName[name]; exists {
		r.mu.RUnlock()
		return def
	}
	r.mu.RUnlock()

	// 需要创建，获取写锁
	r.mu.Lock()
	defer r.mu.Unlock()

	// 双重检查
	if def, exists := r.byName[name]; exists {
		return def
	}

	for {
		if _, exists := r.byCode[r.autoCode]; !exists {
			break
		}
		r.autoCode++
	}

	code := r.autoCode
	r.autoCode++

	def := &Definition{Code: code, Name: name}
	r.byCode[code] = def
	r.byName[name] = def

	return def
}

// Export 导出当前注册表中的所有错误定义信息。
func (r *Registry) Export() []ErrorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ErrorInfo, 0, len(r.byCode))
	for _, def := range r.byCode {
		info := ErrorInfo{
			Code: def.Code,
			Name: def.Name,
		}
		if def.Class != nil {
			info.Class = def.Class.Name
		}
		infos = append(infos, info)
	}
	return infos
}

// Register 在全局注册表中注册错误定义。
func Register(code int, name string) *Definition {
	return global.Register(code, name)
}

// NewDefinition 在全局注册表中创建或返回同名错误定义。
func NewDefinition(name string) *Definition {
	return global.New(name)
}

// New 创建一个带当前定义的错误实例。
func (d *Definition) New(msg string) error {
	return &codedError{def: d, message: msg}
}

// Newf 按格式化字符串创建错误实例。
func (d *Definition) Newf(format string, args ...interface{}) error {
	return d.New(fmt.Sprintf(format, args...))
}

// Wrap 使用当前定义包装底层错误。
func (d *Definition) Wrap(cause error, msg string) error {
	return &codedError{def: d, message: msg, cause: cause}
}

// Wrapf 使用格式化消息包装底层错误。
func (d *Definition) Wrapf(cause error, format string, args ...interface{}) error {
	return d.Wrap(cause, fmt.Sprintf(format, args...))
}

// WithHTTP 返回附带 HTTP 状态码的新定义副本。
func (d *Definition) WithHTTP(code int) *Definition {
	if d == nil {
		return nil
	}
	copied := *d
	copied.HTTP = code
	return &copied
}

// toCodedError 将 error 转换为 *codedError，失败返回 nil。
func toCodedError(err error) *codedError {
	ce, ok := err.(*codedError)
	if !ok || ce == nil || ce.def == nil {
		return nil
	}
	return ce
}

// Code 返回 codedError 的错误码，非 codedError 返回 0。
func Code(err error) int {
	if ce := toCodedError(err); ce != nil {
		return ce.def.Code
	}
	return 0
}

// Name 返回 codedError 的错误名，非 codedError 返回空字符串。
func Name(err error) string {
	if ce := toCodedError(err); ce != nil {
		return ce.def.Name
	}
	return ""
}

// HTTPCode 返回 codedError 对应的 HTTP 状态码，默认 500。
func HTTPCode(err error) int {
	if ce := toCodedError(err); ce != nil {
		if ce.def.HTTP != 0 {
			return ce.def.HTTP
		}
		if ce.def.Class != nil && ce.def.Class.HTTP != 0 {
			return ce.def.Class.HTTP
		}
	}
	return http.StatusInternalServerError
}

// Export 导出全局注册表中的所有错误定义信息。
func Export() []ErrorInfo {
	return global.Export()
}
