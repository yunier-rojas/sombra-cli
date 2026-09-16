package sombra

import (
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestProcessorProcessFileReplacesWholeContent(t *testing.T) {
	processor := NewProcessor()

	content := []byte("old content\nwith many lines\n")
	got, err := processor.ProcessFile(content, entities.MapList{{Key: "file:", Value: "new content\n"}}, nil, true)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	if want := "new content\n"; string(got) != want {
		t.Fatalf("ProcessFile() = %q, want %q", got, want)
	}
}

func TestFileDirectiveWinsOverOtherMappings(t *testing.T) {
	processor := NewProcessor()

	mapping := entities.MapList{
		{Key: "word:sombra", Value: "app"},
		{Key: "file:", Value: "replaced"},
	}
	got, err := processor.ProcessFile([]byte("run sombra"), mapping, nil, true)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	if want := "replaced"; string(got) != want {
		t.Fatalf("ProcessFile() = %q, want %q", got, want)
	}
}

func TestFileDirectiveEmptyReplacement(t *testing.T) {
	processor := NewProcessor()

	got, err := processor.ProcessFile([]byte("something"), entities.MapList{{Key: "file:", Value: ""}}, nil, true)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ProcessFile() = %q, want empty", got)
	}
}

func TestFileDirectiveIsIgnoredForFragmentsAndStrings(t *testing.T) {
	processor := NewProcessor()

	fragment := []byte("+  \"old\": true")
	if got := processor.ProcessContent(fragment, entities.MapList{{Key: "file:", Value: "nope"}}); string(got) != string(fragment) {
		t.Fatalf("ProcessContent() = %q, want %q", got, fragment)
	}

	if got := processor.ProcessString("src/file.go", entities.MapList{{Key: "file:", Value: "nope"}}); got != "src/file.go" {
		t.Fatalf("ProcessString() = %q, want %q", got, "src/file.go")
	}
}
