package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJoinUnder(t *testing.T) {
	base := filepath.Join(t.TempDir(), "tmpl")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := JoinUnder(base, "a/b.tpl")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "b.tpl" {
		t.Fatalf("got %q", got)
	}
	_, err = JoinUnder(base, "../escape")
	if err == nil {
		t.Fatal("expected error for escape")
	}
}
