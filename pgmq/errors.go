package pgmq

import "errors"

var (
	ErrMissingDB        = errors.New("db 不能为空")
	ErrMissingQueue     = errors.New("queue 名称不能为空")
	ErrInvalidQueue     = errors.New("queue 名称不合法")
	ErrInvalidDelay     = errors.New("延迟时间不能为负")
	ErrInvalidQuantity  = errors.New("读取数量必须大于0")
	ErrInvalidConfig    = errors.New("配置无效")
	ErrExtensionMissing = errors.New("pgmq 扩展不存在")
	ErrDecodeMessage    = errors.New("消息解码失败")
)
