package httpclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Response HTTP响应
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
	Response   *http.Response
	Request    *http.Request
	Duration   time.Duration
}

// JSON 解析响应为JSON
func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// String 获取响应字符串
func (r *Response) String() string {
	return string(r.Body)
}

// Bytes 获取响应字节
func (r *Response) Bytes() []byte {
	return r.Body
}

// IsSuccess 检查是否为成功响应 (仅2xx)
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsOK 检查是否为OK响应 (2xx + 3xx)
func (r *Response) IsOK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 400
}

// IsRedirect 检查是否为重定向响应
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode < 400
}

// IsClientError 检查是否为客户端错误
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError 检查是否为服务器错误
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500
}

// IsError 检查是否为错误响应 (4xx + 5xx)
func (r *Response) IsError() bool {
	return r.StatusCode >= 400
}

// IsInformational 检查是否为信息性响应
func (r *Response) IsInformational() bool {
	return r.StatusCode >= 100 && r.StatusCode < 200
}

// Error 获取错误信息
func (r *Response) Error() string {
	if r.IsError() {
		return fmt.Sprintf("HTTP %d: %s", r.StatusCode, r.String())
	}
	return ""
}
