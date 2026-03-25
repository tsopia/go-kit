package httpserver

import "errors"

// ErrInvalidConfig 表示服务器配置不合法。
var ErrInvalidConfig = errors.New("invalid config")

// ErrWSSessionClosed 表示 WebSocket session 已关闭，不能再发送消息。
var ErrWSSessionClosed = errors.New("websocket session closed")
