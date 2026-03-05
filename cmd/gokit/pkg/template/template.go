package template

import (
	"bytes"
	"fmt"
	"text/template"
)

type Data struct {
	Name         string
	Module       string
	GoKitModule  string
	GoKitVersion string
	Features     []string
}

func Render(name, tmpl string, data Data) (string, error) {
	t := template.New(name)
	t, err := t.Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
