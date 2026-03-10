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
	decoder       func(*gin.Context, any) error
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

// WithDecoder 覆盖默认请求解码器。
func WithDecoder[Req any](decoder func(*gin.Context, *Req) error) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.decoder = func(c *gin.Context, target any) error {
			req, ok := target.(*Req)
			if !ok {
				return fmt.Errorf("decode target type mismatch")
			}

			return decoder(c, req)
		}
	}
}

func newHandlerConfig() handlerConfig {
	return handlerConfig{
		successStatus: http.StatusOK,
		decoder: func(c *gin.Context, target any) error {
			return c.ShouldBindJSON(target)
		},
		encoder: func(c *gin.Context, status int, resp any) {
			c.JSON(status, resp)
		},
	}
}

// Handle 将强类型业务函数适配成通用 HTTP handler。
func Handle[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption) gin.HandlerFunc {
	cfg := newHandlerConfig()

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(c *gin.Context) {
		var req Req
		if err := cfg.decoder(c, &req); err != nil {
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

// HandleJSON 将强类型业务函数适配成 JSON HTTP handler。
func HandleJSON[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption) gin.HandlerFunc {
	combined := make([]HandlerOption, 0, len(opts)+1)
	combined = append(combined, WithDecoder(DecodeJSON[Req]()))
	combined = append(combined, opts...)

	return Handle(fn, combined...)
}

// DecodeJSON 使用 JSON body 填充请求对象。
func DecodeJSON[Req any]() func(*gin.Context, *Req) error {
	return func(c *gin.Context, req *Req) error {
		return c.ShouldBindJSON(req)
	}
}

// DecodeQuery 使用 query string 填充请求对象。
func DecodeQuery[Req any]() func(*gin.Context, *Req) error {
	return func(c *gin.Context, req *Req) error {
		return c.ShouldBindQuery(req)
	}
}

// DecodeURI 使用 URI 参数填充请求对象。
func DecodeURI[Req any]() func(*gin.Context, *Req) error {
	return func(c *gin.Context, req *Req) error {
		return c.ShouldBindUri(req)
	}
}

// ComposeDecoder 顺序执行多个 decoder。
func ComposeDecoder[Req any](decoders ...func(*gin.Context, *Req) error) func(*gin.Context, *Req) error {
	return func(c *gin.Context, req *Req) error {
		for _, decoder := range decoders {
			if decoder == nil {
				continue
			}
			if err := decoder(c, req); err != nil {
				return err
			}
		}

		return nil
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
