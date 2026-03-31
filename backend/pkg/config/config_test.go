package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureDownloadsCovletDir_UsesHome(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	t.Cleanup(func() { _ = os.Setenv("HOME", oldHome) })

	dir, err := EnsureDownloadsCovletDir()
	if err != nil {
		t.Fatalf("EnsureDownloadsCovletDir error: %v", err)
	}
	expected := filepath.Join(tmp, "Downloads", "covlet")
	if dir != expected {
		t.Fatalf("unexpected dir: got %q want %q", dir, expected)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}

	_ = runtime.GOOS
}

func TestEnsureTemplatesDir_WithSetMainDir(t *testing.T) {
	base := t.TempDir()
	if err := SetMainDir(base); err != nil {
		t.Fatalf("SetMainDir: %v", err)
	}
	dir, err := EnsureTemplatesDir()
	if err != nil {
		t.Fatalf("EnsureTemplatesDir error: %v", err)
	}
	if dir != filepath.Join(base, "templates") {
		t.Fatalf("unexpected templates dir: %q", dir)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("templates dir not created properly")
	}
}
