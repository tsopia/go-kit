package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// StructTool 是一个泛型工具，用于让模型生成指定结构体。
//
// 它利用 Tool Call 的 JSON Schema 约束模型输出格式：
//   - 模型按 Schema 生成 JSON 参数
//   - Invoke 内部做 json.Unmarshal 校验
//   - 成功 → 返回合法 JSON
//   - 失败 → 返回 error，触发自动重试
//
// 配合 ToolChoiceForced + ToolReturnDirectly 使用，实现「结构化输出提取」：
//
//	type JD struct {
//	    Title        string   `json:"title"`
//	    Requirements []string `json:"requirements"`
//	}
//
//	tool := llm.NewStructTool[JD]("generate_jd", "生成职位描述")
//
//	forced := schema.ToolChoiceForced
//	agent, _ := llm.NewAgent(ctx, llm.AgentConfig{
//	    ModelConfig:        cfg,
//	    InvokableTools:     []llm.InvokableTool{tool},
//	    ToolChoice:         &forced,
//	    ToolReturnDirectly: map[string]struct{}{"generate_jd": {}},
//	    SystemPrompt:       "根据用户需求生成职位描述。",
//	})
//
//	msg, _ := agent.Generate(ctx, messages)
//	var jd JD
//	json.Unmarshal([]byte(msg.Content), &jd)
type StructTool[T any] struct {
	name string
	desc string
	info *schema.ToolInfo
}

// NewStructTool 创建一个结构化输出提取工具。
// 自动从 T 的 json tag 生成 ToolInfo 的参数定义。
func NewStructTool[T any](name, desc string) *StructTool[T] {
	params := structToParams[T]()
	return &StructTool[T]{
		name: name,
		desc: desc,
		info: &schema.ToolInfo{
			Name:        name,
			Desc:        desc,
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		},
	}
}

func (s *StructTool[T]) Info() *schema.ToolInfo {
	return s.info
}

func (s *StructTool[T]) Invoke(_ context.Context, args string) (string, error) {
	var v T
	if err := json.Unmarshal([]byte(args), &v); err != nil {
		return "", fmt.Errorf("参数校验失败: %w。请严格按照 JSON Schema 重新生成参数", err)
	}
	// 重新序列化，确保输出格式标准化
	out, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("序列化失败: %w", err)
	}
	return string(out), nil
}

// structToParams 从结构体 T 的 json tag 自动生成参数定义。
// 支持基本类型（string, int, float, bool）和嵌套结构。
func structToParams[T any]() map[string]*schema.ParameterInfo {
	var zero T
	t := reflect.TypeOf(zero)
	params, _ := typeToParams(t)
	return params
}

// typeToParams 返回 (MapParams, SelfInfo)。
// MapParams 仅当 self 为 Object 且为结构体时有值。
// SelfInfo 始终包含自身的类型描述。
func typeToParams(t reflect.Type) (map[string]*schema.ParameterInfo, *schema.ParameterInfo) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 如果是 Slice/Array，返回 ElemInfo
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		elemType := t.Elem()
		_, elemInfo := typeToParams(elemType) // 递归处理元素类型

		return nil, &schema.ParameterInfo{
			Type:     schema.Array,
			ElemInfo: elemInfo,
		}
	}

	// 如果是结构体，生成 SubParams
	if t.Kind() == reflect.Struct {
		params := make(map[string]*schema.ParameterInfo)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			// 处理根据 json tag 获取字段名
			jsonName := jsonTag
			if idx := findChar(jsonName, ','); idx >= 0 {
				jsonName = jsonName[:idx]
			}

			// 嵌入结构体且没有显式 JSON tag：需要扁平化合并
			if field.Anonymous && jsonName == "" {
				embeddedParams, _ := typeToParams(field.Type)
				// embeddedParams 是嵌入结构体的 map[string]*ParameterInfo
				for k, v := range embeddedParams {
					params[k] = v
				}
				continue
			}

			if jsonName == "" {
				jsonName = field.Name
			}

			desc := field.Tag.Get("desc")
			if desc == "" {
				desc = field.Name
			}

			required := field.Tag.Get("required") == "true"

			var enum []string
			if enumStr := field.Tag.Get("enum"); enumStr != "" {
				enum = strings.Split(enumStr, ",")
			}

			// 递归处理字段类型
			_, fieldInfo := typeToParams(field.Type)

			// 补充当前字段的描述信息（覆盖类型推导的默认信息）
			fieldInfo.Desc = desc
			fieldInfo.Required = required
			fieldInfo.Enum = enum

			params[jsonName] = fieldInfo
		}

		return params, &schema.ParameterInfo{
			Type:      schema.Object,
			SubParams: params,
		}
	}

	// 基本类型
	return nil, &schema.ParameterInfo{
		Type: goTypeToSchemaType(t),
	}
}

func goTypeToSchemaType(t reflect.Type) schema.DataType {
	switch t.Kind() {
	case reflect.String:
		return schema.String
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return schema.Integer
	case reflect.Float32, reflect.Float64:
		return schema.Number
	case reflect.Bool:
		return schema.Boolean
	case reflect.Slice, reflect.Array:
		return schema.Array
	default:
		return schema.Object
	}
}

func findChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
