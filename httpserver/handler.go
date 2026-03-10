package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type handlerErrorKind int

const (
	handlerErrorKindDecode handlerErrorKind = iota + 1
	handlerErrorKindValidate
)

type handlerRequestError struct {
	kind handlerErrorKind
	err  error
}

func (e *handlerRequestError) Error() string {
	return e.err.Error()
}

func (e *handlerRequestError) Unwrap() error {
	return e.err
}

// HandlerFunc 描述强类型请求的业务处理函数。
type HandlerFunc[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)

// ErrorMapper 负责把业务错误映射成 HTTP 响应。
type ErrorMapper func(err error) (status int, body any)

// HandlerOption 描述 handler 的可选配置。
type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	successStatus int
	encoder       func(*gin.Context, int, any)
	errorMapper   ErrorMapper
}

// WithSuccessStatus 覆盖成功响应状态码。
func WithSuccessStatus(status int) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.successStatus = status
	}
}

// WithErrorMapper 为业务错误提供自定义映射。
func WithErrorMapper(mapper ErrorMapper) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.errorMapper = mapper
	}
}

// WithEncoder 覆盖成功响应编码器。
func WithEncoder(encoder func(*gin.Context, int, any)) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.encoder = encoder
	}
}

// HandleJSON 将强类型业务函数适配成 JSON HTTP handler。
func HandleJSON[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption) gin.HandlerFunc {
	cfg := handlerConfig{
		successStatus: http.StatusOK,
		encoder: func(c *gin.Context, status int, resp any) {
			c.JSON(status, resp)
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(c *gin.Context) {
		var req Req
		if err := c.ShouldBindJSON(&req); err != nil {
			renderHandlerError(c, cfg, &handlerRequestError{
				kind: handlerErrorKindDecode,
				err:  fmt.Errorf("decode request: %w", err),
			})
			return
		}

		ctx := ContextFromGin(c)
		if err := validateRequest(ctx, req); err != nil {
			renderHandlerError(c, cfg, &handlerRequestError{
				kind: handlerErrorKindValidate,
				err:  fmt.Errorf("validate request: %w", err),
			})
			return
		}

		resp, err := fn(ctx, req)
		if err != nil {
			renderHandlerError(c, cfg, err)
			return
		}

		cfg.encoder(c, cfg.successStatus, resp)
	}
}

type contextValidator interface {
	Validate(context.Context) error
}

type validator interface {
	Validate() error
}

func validateRequest[Req any](ctx context.Context, req Req) error {
	if v, ok := any(req).(contextValidator); ok {
		return v.Validate(ctx)
	}
	if v, ok := any(req).(validator); ok {
		return v.Validate()
	}

	return nil
}

func renderHandlerError(c *gin.Context, cfg handlerConfig, err error) {
	var requestErr *handlerRequestError
	if errors.As(err, &requestErr) {
		switch requestErr.kind {
		case handlerErrorKindDecode:
			c.JSON(http.StatusBadRequest, gin.H{"error": requestErr.Error()})
			return
		case handlerErrorKindValidate:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": requestErr.Error()})
			return
		}
	}

	if cfg.errorMapper != nil {
		status, body := cfg.errorMapper(err)
		c.JSON(status, body)
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
