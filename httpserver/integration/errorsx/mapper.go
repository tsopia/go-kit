package errorsx

import (
	"strings"

	kiterrors "github.com/tsopia/go-kit/errors"
	"github.com/tsopia/go-kit/httpserver"
)

const internalServerErrorMessage = "internal server error"

// ResponseBody 描述 errors 包桥接到 HTTP 响应时使用的默认结构。
type ResponseBody struct {
	Code    int            `json:"code"`
	Name    string         `json:"name"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Response 将 errors 包的业务错误映射为 HTTP 状态码和响应体。
func Response(err error) (int, ResponseBody) {
	code := kiterrors.Code(err)
	name := kiterrors.Name(err)
	if code == 0 || name == "" {
		return kiterrors.HTTPCode(kiterrors.Internal.New(internalServerErrorMessage)), ResponseBody{
			Code:    kiterrors.Internal.Code,
			Name:    kiterrors.Internal.Name,
			Message: internalServerErrorMessage,
		}
	}

	return kiterrors.HTTPCode(err), ResponseBody{
		Code:    code,
		Name:    name,
		Message: messageFromError(err, name),
	}
}

// Mapper 返回可直接挂到 typed handler 的 errors 到 HTTP 响应映射器。
func Mapper() httpserver.ErrorMapper {
	return func(err error) (int, any) {
		status, body := Response(err)
		return status, body
	}
}

func messageFromError(err error, name string) string {
	if err == nil {
		return internalServerErrorMessage
	}

	message := err.Error()
	prefix := "[" + name + "]"
	switch {
	case strings.HasPrefix(message, prefix+" "):
		return strings.TrimPrefix(message, prefix+" ")
	case message == prefix:
		return name
	default:
		return message
	}
}
