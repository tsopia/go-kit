package tool

import (
	"context"
	"encoding/json"
)

type JSONType string

const (
	JSONTypeString  JSONType = "string"
	JSONTypeNumber  JSONType = "number"
	JSONTypeBool    JSONType = "bool"
	JSONTypeObject  JSONType = "object"
	JSONTypeArray   JSONType = "array"
	JSONTypeUnknown JSONType = "unknown"
)

type ArgSchema struct {
	Required   []string
	Properties map[string]JSONType
	Strict     bool
}

type InvokableTool interface {
	Name() string
	Schema() ArgSchema
	Invoke(ctx context.Context, args json.RawMessage) (any, error)
}
