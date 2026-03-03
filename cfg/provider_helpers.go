package cfg

import "fmt"

// checkDefaultCount 检查默认值参数个数
func checkDefaultCount[T any](key string, defaults []T) error {
	if len(defaults) > 1 {
		return fmt.Errorf("%w: key %q expects at most one default value, got %d", ErrTypeMismatch, key, len(defaults))
	}
	return nil
}
