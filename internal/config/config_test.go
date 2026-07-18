package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	dir := filepath.Join(base, "manul")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDirHonorsXDGConfigHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "manul"); dir != want {
		t.Errorf("Dir() = %q, want %q", dir, want)
	}
}

func TestLoadAbsentFileReturnsDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg != Defaults() {
		t.Errorf("Load() = %+v, want defaults %+v", cfg, Defaults())
	}
}

func TestLoadDefaultsValues(t *testing.T) {
	d := Defaults()
	if d.Theme != "auto" || d.MaxWidth != 100 || d.Mouse || d.StylePath != "" {
		t.Errorf("unexpected defaults: %+v", d)
	}
}

func TestLoadPartialFileMergesOverDefaults(t *testing.T) {
	writeConfig(t, "theme = \"dark\"\nmouse = true\n")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "dark")
	}
	if cfg.Mouse != true {
		t.Error("Mouse = false, want true")
	}
	if cfg.MaxWidth != 100 {
		t.Errorf("MaxWidth = %d, want default 100", cfg.MaxWidth)
	}
	if cfg.StylePath != "" {
		t.Errorf("StylePath = %q, want empty default", cfg.StylePath)
	}
}

func TestLoadFullFile(t *testing.T) {
	writeConfig(t, `
theme = "light"
max_width = 80
style_path = "/tmp/style.json"
mouse = true
`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	want := Config{Theme: "light", MaxWidth: 80, StylePath: "/tmp/style.json", Mouse: true}
	if cfg != want {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoadInvalidTOMLReturnsError(t *testing.T) {
	writeConfig(t, "theme = not quoted\n")
	if _, err := Load(); err == nil {
		t.Fatal("Load() with invalid TOML: expected error, got nil")
	}
}
