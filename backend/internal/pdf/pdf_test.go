package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestTextToPDF_Basic(t *testing.T) {
	b, err := TextToPDF("Hello", "First line.\n\nSecond paragraph.")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 500 {
		t.Fatalf("pdf too small: %d bytes", len(b))
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("expected PDF header")
	}
}

func TestTextToPDF_Empty(t *testing.T) {
	_, err := TextToPDF("", "")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestTextToPDF_TooLarge(t *testing.T) {
	huge := strings.Repeat("x", maxPDFInputBytes+1)
	_, err := TextToPDF("", huge)
	if err == nil {
		t.Fatal("expected error for oversized content")
	}
}
