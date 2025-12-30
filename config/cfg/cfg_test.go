package cfg

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type sampleConfig struct {
	App struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"app"`
	Flags struct {
		Debug bool `mapstructure:"debug"`
	} `mapstructure:"flags"`
	Limits struct {
		Timeout string `mapstructure:"timeout"`
	} `mapstructure:"limits"`
}

func writeConfig(t *testing.T, dir string) string {
	t.Helper()
	configFile := filepath.Join(dir, "config.yml")
	content := `app:
  name: test-app
  port: 9000
flags:
  debug: true
limits:
  timeout: 1s
strings:
  list:
    - a
    - b
maps:
  values:
    key: value
`
	if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return configFile
}

func TestLoadAndBind(t *testing.T) {
	tempDir := t.TempDir()
	configFile := writeConfig(t, tempDir)

	var cfgStruct sampleConfig
	manager, err := Load(&cfgStruct, configFile)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfgStruct.App.Name != "test-app" || cfgStruct.App.Port != 9000 {
		t.Fatalf("unexpected struct content: %+v", cfgStruct.App)
	}

	name, err := manager.GetString("app.name")
	if err != nil {
		t.Fatalf("read string failed: %v", err)
	}
	if name != "test-app" {
		t.Fatalf("unexpected name: %s", name)
	}

	port, err := GetInt("app.port")
	if err != nil {
		t.Fatalf("read int failed: %v", err)
	}
	if port != 9000 {
		t.Fatalf("unexpected port: %d", port)
	}
}

func TestDefaultsAndMissing(t *testing.T) {
	tempDir := t.TempDir()
	configFile := writeConfig(t, tempDir)

	if _, err := Load(nil, configFile); err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	fallback, err := GetString("missing.key", "default")
	if err != nil {
		t.Fatalf("expected default without error: %v", err)
	}
	if fallback != "default" {
		t.Fatalf("unexpected default: %s", fallback)
	}

	_, err = GetString("missing.key")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTypeMismatch(t *testing.T) {
	tempDir := t.TempDir()
	configFile := writeConfig(t, tempDir)

	if _, err := Load(nil, configFile); err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if _, err := GetInt("app.name"); !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("expected ErrTypeMismatch, got %v", err)
	}
}

func TestCompositeTypes(t *testing.T) {
	tempDir := t.TempDir()
	configFile := writeConfig(t, tempDir)

	if _, err := Load(nil, configFile); err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	slice, err := GetStringSlice("strings.list")
	if err != nil {
		t.Fatalf("string slice read failed: %v", err)
	}
	if len(slice) != 2 || slice[0] != "a" || slice[1] != "b" {
		t.Fatalf("unexpected slice: %#v", slice)
	}

	m, err := GetStringMap("maps.values")
	if err != nil {
		t.Fatalf("string map read failed: %v", err)
	}
	if m["key"] != "value" {
		t.Fatalf("unexpected map content: %#v", m)
	}
}

func TestDurationAndTime(t *testing.T) {
	tempDir := t.TempDir()
	configFile := writeConfig(t, tempDir)

	if _, err := Load(nil, configFile); err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	duration, err := GetDuration("limits.timeout")
	if err != nil {
		t.Fatalf("duration read failed: %v", err)
	}
	if duration != time.Second {
		t.Fatalf("unexpected duration: %v", duration)
	}

	// when not set, allow default
	ts, err := GetTime("limits.deadline", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("time with default failed: %v", err)
	}
	if ts.Year() != 2024 {
		t.Fatalf("unexpected default time: %v", ts)
	}
}

func TestNotLoadedGuard(t *testing.T) {
	defaultMutex.Lock()
	defaultManager = nil
	defaultMutex.Unlock()

	if _, err := GetString("app.name"); !errors.Is(err, ErrNotLoaded) {
		t.Fatalf("expected ErrNotLoaded, got %v", err)
	}
}
