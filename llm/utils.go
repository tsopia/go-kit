package llm

import (
	"log"

	"github.com/cloudwego/eino/components/tool"
)

// CombineTools 合并单个工具或工具列表。
// 支持 tool.BaseTool 和 []tool.BaseTool 类型。
// 对于不支持的类型，会打印警告并忽略。
func CombineTools(args ...interface{}) []tool.BaseTool {
	var results []tool.BaseTool
	for _, arg := range args {
		switch v := arg.(type) {
		case tool.BaseTool:
			results = append(results, v)
		case []tool.BaseTool:
			results = append(results, v...)
		case nil:
			// ignore
		default:
			log.Printf("CombineTools: unsupported type %T, ignored", arg)
		}
	}
	return results
}
