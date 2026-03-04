package errors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestExportRegistry(t *testing.T) {
	r := NewRegistry()

	base := r.Register(1001, "BASE")
	child := r.Register(1002, "CHILD")
	child.Class = base
	auto := r.New("AUTO")

	infos := r.Export()

	if len(infos) != 3 {
		t.Fatalf("expected 3 items, got %d", len(infos))
	}

	byName := make(map[string]ErrorInfo, len(infos))
	for _, info := range infos {
		byName[info.Name] = info
	}

	if got, ok := byName["BASE"]; !ok {
		t.Fatalf("BASE not found in export")
	} else {
		if got.Code != 1001 || got.Class != "" {
			t.Fatalf("unexpected BASE export: %+v", got)
		}
	}

	if got, ok := byName["CHILD"]; !ok {
		t.Fatalf("CHILD not found in export")
	} else {
		if got.Code != 1002 || got.Class != "BASE" {
			t.Fatalf("unexpected CHILD export: %+v", got)
		}
	}

	if got, ok := byName["AUTO"]; !ok {
		t.Fatalf("AUTO not found in export")
	} else {
		if got.Code != auto.Code || got.Class != "" {
			t.Fatalf("unexpected AUTO export: %+v", got)
		}
	}
}

func TestExportGlobal(t *testing.T) {
	old := global
	global = NewRegistry()
	t.Cleanup(func() { global = old })

	base := Register(2001, "G_BASE")
	child := Register(2002, "G_CHILD")
	child.Class = base

	infos := Export()
	if len(infos) != 2 {
		t.Fatalf("expected 2 items, got %d", len(infos))
	}

	byName := make(map[string]ErrorInfo, len(infos))
	for _, info := range infos {
		byName[info.Name] = info
	}

	if got := byName["G_BASE"]; got.Code != 2001 || got.Class != "" {
		t.Fatalf("unexpected G_BASE export: %+v", got)
	}
	if got := byName["G_CHILD"]; got.Code != 2002 || got.Class != "G_BASE" {
		t.Fatalf("unexpected G_CHILD export: %+v", got)
	}
}

func TestGenerateDoc(t *testing.T) {
	if os.Getenv("GEN_ERR_DOC") != "1" {
		t.Skip("skip doc generation unless GEN_ERR_DOC=1")
	}

	infos := Export()
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Code == infos[j].Code {
			return infos[i].Name < infos[j].Name
		}
		return infos[i].Code < infos[j].Code
	})

	var b strings.Builder
	b.WriteString("# Error Codes\n\n")
	b.WriteString("| Code | Name | Class |\n")
	b.WriteString("|------|------|-------|\n")

	for _, info := range infos {
		class := info.Class
		if class == "" {
			class = "-"
		}
		fmt.Fprintf(&b, "| %d | %s | %s |\n", info.Code, info.Name, class)
	}

	out := filepath.Join("..", "docs", "error-codes.md")
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
}
