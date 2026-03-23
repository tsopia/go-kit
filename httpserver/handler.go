package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

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

// ErrorResponse 描述 typed handler 的默认错误响应结构。
type ErrorResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// HandlerFunc 描述强类型请求的业务处理函数。
type HandlerFunc[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)

// ErrorMapper 负责把业务错误映射成 HTTP 响应。
type ErrorMapper func(err error) (status int, body any)

// HTTPError 描述带有 HTTP 语义的业务错误。
type HTTPError interface {
	error
	StatusCode() int
	ErrorCode() string
	ErrorMessage() string
	ErrorDetails() map[string]any
}

// HandlerOption 描述 handler 的可选配置。
type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	successStatus int
	decoder       func(*gin.Context, any) error
	validators    []func(context.Context, any) error
	encoder       func(*gin.Context, int, any)
	errorMapper   ErrorMapper
}

// ValidationField 描述字段级校验错误。
type ValidationField struct {
	Field   string `json:"field"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// ValidationError 描述结构化请求校验错误。
type ValidationError struct {
	Message string            `json:"message"`
	Fields  []ValidationField `json:"fields,omitempty"`
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if len(e.Fields) > 0 {
		return e.Fields[0].Message
	}

	return "validation failed"
}

// RequestValidator 描述 typed handler 的显式请求校验器。
type RequestValidator[Req any] func(context.Context, Req) error

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

// WithValidators 为 typed handler 追加显式请求校验器。
func WithValidators[Req any](validators ...RequestValidator[Req]) HandlerOption {
	return func(cfg *handlerConfig) {
		for _, validator := range validators {
			if validator == nil {
				continue
			}

			cfg.validators = append(cfg.validators, func(ctx context.Context, req any) error {
				typedReq, ok := req.(Req)
				if !ok {
					return fmt.Errorf("validator request type mismatch")
				}

				return validator(ctx, typedReq)
			})
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
		if err := applyValidators(ctx, req, cfg.validators); err != nil {
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

// HandleQuery 将强类型业务函数适配成 query string handler。
func HandleQuery[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption) gin.HandlerFunc {
	combined := make([]HandlerOption, 0, len(opts)+1)
	combined = append(combined, WithDecoder(DecodeQuery[Req]()))
	combined = append(combined, opts...)

	return Handle(fn, combined...)
}

// HandleURI 将强类型业务函数适配成 URI 参数 handler。
func HandleURI[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption) gin.HandlerFunc {
	combined := make([]HandlerOption, 0, len(opts)+1)
	combined = append(combined, WithDecoder(DecodeURI[Req]()))
	combined = append(combined, opts...)

	return Handle(fn, combined...)
}

// HandleQueryURI 将强类型业务函数适配成 URI + query 组合解码 handler。
func HandleQueryURI[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption) gin.HandlerFunc {
	combined := make([]HandlerOption, 0, len(opts)+1)
	combined = append(combined, WithDecoder(ComposeDecoder(
		DecodeURI[Req](),
		DecodeQuery[Req](),
	)))
	combined = append(combined, opts...)

	return Handle(fn, combined...)
}

// HandleUpload 将强类型业务函数适配成上传 handler。
// 自动清除 ReadDeadline 和 WriteDeadline，限制请求体大小。
func HandleUpload[Req any, Resp any](
	fn HandlerFunc[Req, Resp],
	maxBytes int64,
	opts ...HandlerOption,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 清除 deadline
		rc := http.NewResponseController(c.Writer)
		_ = rc.SetReadDeadline(time.Time{})
		_ = rc.SetWriteDeadline(time.Time{})

		// 2. 限制 body 大小
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		// 3. 走正常的 Handle 逻辑
		Handle(fn, opts...)(c)
	}
}

// HandleForm 将强类型业务函数适配成 form handler。
// 使用 Gin 的 ShouldBind，支持 JSON/Form/Multipart 自动识别。
func HandleForm[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption) gin.HandlerFunc {
	combined := make([]HandlerOption, 0, len(opts)+1)
	combined = append(combined, WithDecoder(DecodeForm[Req]()))
	combined = append(combined, opts...)

	return Handle(fn, combined...)
}

// ActionFunc 描述无响应体的业务处理函数。
type ActionFunc[Req any] func(ctx context.Context, req Req) error

// HandleNoContent 将无响应体业务函数适配成 HTTP handler（默认 204）。
func HandleNoContent[Req any](fn ActionFunc[Req], opts ...HandlerOption) gin.HandlerFunc {
	adapted := func(ctx context.Context, req Req) (struct{}, error) {
		return struct{}{}, fn(ctx, req)
	}

	combined := make([]HandlerOption, 0, len(opts)+2)
	combined = append(combined, WithSuccessStatus(http.StatusNoContent))
	combined = append(combined, WithEncoder(func(c *gin.Context, status int, _ any) {
		c.Status(status)
	}))
	combined = append(combined, opts...)

	return Handle(adapted, combined...)
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

// DecodeForm 使用 Gin 的 ShouldBind 填充请求对象（支持 JSON/Form/Multipart 自动识别）。
func DecodeForm[Req any]() func(*gin.Context, *Req) error {
	return func(c *gin.Context, req *Req) error {
		return c.ShouldBind(req)
	}
}

// DecodeHeader 使用 Header 填充请求对象。
func DecodeHeader[Req any]() func(*gin.Context, *Req) error {
	return func(c *gin.Context, req *Req) error {
		return c.ShouldBindHeader(req)
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

func applyValidators[Req any](ctx context.Context, req Req, validators []func(context.Context, any) error) error {
	for _, validator := range validators {
		if validator == nil {
			continue
		}
		if err := validator(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

func renderHandlerError(c *gin.Context, cfg handlerConfig, err error) {
	var requestErr *handlerRequestError
	if errors.As(err, &requestErr) {
		switch requestErr.kind {
		case handlerErrorKindDecode:
			var maxBytesErr *http.MaxBytesError
			if errors.As(requestErr, &maxBytesErr) {
				c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
					Code:    "request_too_large",
					Message: "request body too large",
				})
				return
			}
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "invalid_request",
				Message: requestErr.Error(),
			})
			return
		case handlerErrorKindValidate:
			var validationErr *ValidationError
			if errors.As(requestErr, &validationErr) {
				resp := ErrorResponse{
					Code:    "validation_failed",
					Message: validationErr.Error(),
				}
				if len(validationErr.Fields) > 0 {
					resp.Details = map[string]any{
						"fields": validationErr.Fields,
					}
				}
				c.JSON(http.StatusUnprocessableEntity, resp)
				return
			}

			root := error(requestErr)
			for {
				unwrapped := errors.Unwrap(root)
				if unwrapped == nil {
					break
				}
				root = unwrapped
			}
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Code:    "validation_failed",
				Message: root.Error(),
			})
			return
		}
	}

	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		resp := ErrorResponse{
			Code:    httpErr.ErrorCode(),
			Message: httpErr.ErrorMessage(),
		}
		if details := httpErr.ErrorDetails(); len(details) > 0 {
			resp.Details = details
		}
		c.JSON(httpErr.StatusCode(), resp)
		return
	}

	if cfg.errorMapper != nil {
		status, body := cfg.errorMapper(err)
		c.JSON(status, body)
		return
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    "internal_error",
		Message: err.Error(),
	})
}
