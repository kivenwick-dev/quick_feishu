package main

import (
	"path/filepath"
	"testing"
)

func TestResolveDataDirEnvOverride(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "volume")
	t.Setenv("QF_DATA_DIR", dir)
	if got := resolveDataDir(); got != dir {
		t.Fatalf("resolveDataDir = %s, want %s", got, dir)
	}
}

func TestResolveDataDirDefault(t *testing.T) {
	t.Setenv("QF_DATA_DIR", "")
	want := filepath.Join(baseDir(), "data")
	if got := resolveDataDir(); got != want {
		t.Fatalf("resolveDataDir = %s, want %s", got, want)
	}
}

func TestResolveConfigPathEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("QF_CONFIG_FILE", path)
	if got := resolveConfigPath(); got != path {
		t.Fatalf("resolveConfigPath = %s, want %s", got, path)
	}
}

func TestResolveConfigPathDefault(t *testing.T) {
	t.Setenv("QF_CONFIG_FILE", "")
	want := filepath.Join(baseDir(), "config.yaml")
	if got := resolveConfigPath(); got != want {
		t.Fatalf("resolveConfigPath = %s, want %s", got, want)
	}
}
