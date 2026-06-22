package utils

// StreamingKey 是流式连接标记在 gin.Context 中的 key。
// 值为 transport 字符串（"sse" / "ws"），空串表示非流式请求。
// 观测型中间件在 c.Next() 之后读取此 key 以正确处理流式连接。
const StreamingKey = "stream"
