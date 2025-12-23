package llm

// FloatPtr 工具函数，便于在配置中声明浮点指针。
func FloatPtr(v float64) *float64 { return &v }

// IntPtr 工具函数，便于在配置中声明整型指针。
func IntPtr(v int) *int { return &v }
