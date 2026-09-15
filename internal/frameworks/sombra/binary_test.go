package sombra

import (
	"bytes"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

// PNG signature followed by a NUL byte in the chunk length.
var pngContent = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d}

func TestIsBinary(t *testing.T) {
	if isBinary([]byte("plain text\n")) {
		t.Fatal("isBinary(text) = true, want false")
	}
	if !isBinary(pngContent) {
		t.Fatal("isBinary(png) = false, want true")
	}
}

func TestProcessFileLeavesBinaryUntouched(t *testing.T) {
	processor := NewProcessor()
	mapping := entities.MapList{
		{Key: `re:PNG`, Value: "jpg"},
		{Key: "file:", Value: "replaced"},
	}

	got, err := processor.ProcessFile(pngContent, mapping, nil, true)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	if !bytes.Equal(got, pngContent) {
		t.Fatalf("ProcessFile() = %q, want byte-identical binary content", got)
	}
}
