package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if cfg.SplitDiff {
		t.Error("SplitDiff should default to false")
	}
	if !cfg.WrapDiff {
		t.Error("WrapDiff should default to true")
	}
}

func TestSaveAndLoad(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Config{
		SplitDiff: true,
		WrapDiff:  true,
	}
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	got := LoadFrom(path)
	if !got.SplitDiff {
		t.Error("SplitDiff should be true")
	}
	if !got.WrapDiff {
		t.Error("WrapDiff should be true")
	}
}

func TestLoad_NoFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	cfg := LoadFrom(path)
	if cfg.SplitDiff {
		t.Error("missing file should return defaults")
	}
	if !cfg.WrapDiff {
		t.Error("missing file should return defaults with WrapDiff=true")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadFrom(path)
	if cfg.SplitDiff {
		t.Error("invalid JSON should return defaults")
	}
	if !cfg.WrapDiff {
		t.Error("invalid JSON should return defaults with WrapDiff=true")
	}
}

func TestSave_CreatesDir(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "config.json")

	cfg := Default()
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo should create dirs: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file should exist: %v", err)
	}
}
