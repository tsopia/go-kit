package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Request HTTP请求构建器
type Request struct {
	client  *Client
	method  string
	url     string
	headers map[string]string
	cookies []*http.Cookie
	body    io.Reader
	bodyErr error
	bodyRaw []byte
	bodySrc io.ReadSeeker
	timeout time.Duration
	ctx     context.Context
	retries int
}

// Header 设置请求头
func (r *Request) Header(key, value string) *Request {
	r.headers[key] = value
	return r
}

// Headers 批量设置请求头
func (r *Request) Headers(headers map[string]string) *Request {
	for key, value := range headers {
		r.headers[key] = value
	}
	return r
}

// Cookie 添加Cookie
func (r *Request) Cookie(cookie *http.Cookie) *Request {
	r.cookies = append(r.cookies, cookie)
	return r
}

// Body 设置请求体
func (r *Request) Body(body io.Reader) *Request {
	r.bodyErr = nil
	r.bodyRaw = nil
	r.bodySrc = nil

	if body == nil {
		r.body = nil
		return r
	}

	if seeker, ok := body.(io.ReadSeeker); ok {
		r.bodySrc = seeker
		r.body = seeker
		return r
	}

	data, err := io.ReadAll(body)
	if err != nil {
		r.bodyErr = fmt.Errorf("读取请求体失败: %w", err)
		return r
	}

	r.bodyRaw = data
	r.body = bytes.NewReader(data)
	return r
}

// JSON 设置JSON请求体
func (r *Request) JSON(data interface{}) *Request {
	jsonData, err := json.Marshal(data)
	if err != nil {
		r.bodyErr = fmt.Errorf("JSON编码失败: %w", err)
		return r
	}
	r.bodyErr = nil
	r.bodySrc = nil
	r.body = bytes.NewBuffer(jsonData)
	r.bodyRaw = jsonData
	r.headers["Content-Type"] = "application/json"
	return r
}

// Form 设置表单请求体
func (r *Request) Form(data url.Values) *Request {
	r.bodyErr = nil
	r.bodySrc = nil
	encoded := data.Encode()
	r.body = strings.NewReader(encoded)
	r.bodyRaw = []byte(encoded)
	r.headers["Content-Type"] = "application/x-www-form-urlencoded"
	return r
}

// Timeout 设置超时时间
func (r *Request) Timeout(timeout time.Duration) *Request {
	r.timeout = timeout
	return r
}

// Context 设置上下文
func (r *Request) Context(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// WithCtx 设置上下文 (Context方法的简洁版本)
func (r *Request) WithCtx(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// Retries 设置重试次数
func (r *Request) Retries(retries int) *Request {
	r.retries = retries
	return r
}

// Do 执行请求
func (r *Request) Do() (*Response, error) {
	if r.bodyErr != nil {
		return nil, r.bodyErr
	}

	// 应用超时
	if r.timeout > 0 {
		ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
		defer cancel()
		r.ctx = ctx
	}

	return r.client.do(r)
}

func (r *Request) prepareBody() (io.Reader, func() (io.ReadCloser, error), error) {
	switch {
	case r.bodySrc != nil:
		if _, err := r.bodySrc.Seek(0, io.SeekStart); err != nil {
			return nil, nil, fmt.Errorf("重置请求体失败: %w", err)
		}

		return r.bodySrc, func() (io.ReadCloser, error) {
			if _, err := r.bodySrc.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}
			return io.NopCloser(r.bodySrc), nil
		}, nil
	case r.bodyRaw != nil:
		return bytes.NewReader(r.bodyRaw), func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(r.bodyRaw)), nil
		}, nil
	case r.body != nil:
		return r.body, nil, nil
	default:
		return nil, nil, nil
	}
}
