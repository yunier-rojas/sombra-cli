package usecases

import (
	"errors"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"go.uber.org/mock/gomock"
)

func TestSombraEngineMatchSkipsDeletePatterns(t *testing.T) {
	ctrl := gomock.NewController(t)
	dirManager := NewMockDirectoryManagerPort(ctrl)
	engine := NewSombraEngineInteractor(dirManager, nil, nil)

	patterns := []*entities.Pattern{{Pattern: "/legacy/**", Delete: true}}
	include, res, err := engine.Match(entities.File("/legacy/old.go"), patterns)
	if err != nil {
		t.Fatalf("Match() unexpected error: %v", err)
	}
	if include {
		t.Fatal("Match() should not include files for tombstone patterns")
	}
	if len(res) != 0 {
		t.Fatalf("Match() returned %d patterns, want 0", len(res))
	}
}

func TestSombraEngineMatchDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	dirManager := NewMockDirectoryManagerPort(ctrl)
	engine := NewSombraEngineInteractor(dirManager, nil, nil)

	file := entities.File("/legacy/old.go")
	patterns := []*entities.Pattern{
		{Pattern: "/keep/**"},
		{Pattern: "/legacy/**", Delete: true},
	}

	dirManager.EXPECT().PathMatch(file, entities.Wildcard("/legacy/**")).Return(true, nil)
	for _, ignore := range engine.alwaysIgnore {
		dirManager.EXPECT().PathMatch(file, ignore).Return(false, nil)
	}

	match, err := engine.MatchDelete(file, patterns)
	if err != nil {
		t.Fatalf("MatchDelete() unexpected error: %v", err)
	}
	if !match {
		t.Fatal("MatchDelete() = false, want true")
	}
}

func TestSombraEngineMatchDeleteNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	dirManager := NewMockDirectoryManagerPort(ctrl)
	engine := NewSombraEngineInteractor(dirManager, nil, nil)

	file := entities.File("/src/main.go")
	patterns := []*entities.Pattern{{Pattern: "/legacy/**", Delete: true}}

	dirManager.EXPECT().PathMatch(file, entities.Wildcard("/legacy/**")).Return(false, nil)

	match, err := engine.MatchDelete(file, patterns)
	if err != nil {
		t.Fatalf("MatchDelete() unexpected error: %v", err)
	}
	if match {
		t.Fatal("MatchDelete() = true, want false")
	}
}

func TestPruneTargetWithoutDeletePatterns(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanner := NewMockDirectoryManagerPort(ctrl)
	files := NewMockFileManagerPort(ctrl)
	engine := NewMockSombraEngineCase(ctrl)

	got, err := pruneTarget(scanner, files, engine, "/target", []*entities.Pattern{{Pattern: "/**/*"}}, false)
	if err != nil {
		t.Fatalf("pruneTarget() unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("pruneTarget() = %v, want nil", got)
	}
}

func TestPruneTargetReportsWithoutPrune(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanner := NewMockDirectoryManagerPort(ctrl)
	files := NewMockFileManagerPort(ctrl)
	engine := NewMockSombraEngineCase(ctrl)

	patterns := []*entities.Pattern{{Pattern: "/legacy/**", Delete: true}}
	scanner.EXPECT().
		ScanTree("/target", []entities.Wildcard{"**/*"}, nil).
		Return(createScanResultChannel([]entities.FileScanResult{
			{File: "/legacy", IsDir: true},
			{File: "/legacy/old.go"},
			{File: "/keep.go"},
		}))
	engine.EXPECT().MatchDelete(entities.File("/legacy"), patterns).Return(true, nil)
	engine.EXPECT().MatchDelete(entities.File("/legacy/old.go"), patterns).Return(true, nil)
	engine.EXPECT().MatchDelete(entities.File("/keep.go"), patterns).Return(false, nil)

	got, err := pruneTarget(scanner, files, engine, "/target", patterns, false)
	if err != nil {
		t.Fatalf("pruneTarget() unexpected error: %v", err)
	}
	want := []string{"/legacy", "/legacy/old.go"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("pruneTarget() = %v, want %v", got, want)
	}
}

func TestPruneTargetRemovesWhenPruneEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanner := NewMockDirectoryManagerPort(ctrl)
	files := NewMockFileManagerPort(ctrl)
	engine := NewMockSombraEngineCase(ctrl)

	patterns := []*entities.Pattern{{Pattern: "/legacy/**", Delete: true}}
	scanner.EXPECT().
		ScanTree("/target", []entities.Wildcard{"**/*"}, nil).
		Return(createScanResultChannel([]entities.FileScanResult{
			{File: "/legacy", IsDir: true},
			{File: "/legacy/old.go"},
		}))
	engine.EXPECT().MatchDelete(entities.File("/legacy"), patterns).Return(true, nil)
	engine.EXPECT().MatchDelete(entities.File("/legacy/old.go"), patterns).Return(true, nil)

	files.EXPECT().Remove("/target", entities.File("/legacy/old.go")).Return(nil)
	files.EXPECT().Remove("/target", entities.File("/legacy")).Return(nil)

	got, err := pruneTarget(scanner, files, engine, "/target", patterns, true)
	if err != nil {
		t.Fatalf("pruneTarget() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("pruneTarget() = %v, want 2 entries", got)
	}
}

func TestPruneTargetRemoveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanner := NewMockDirectoryManagerPort(ctrl)
	files := NewMockFileManagerPort(ctrl)
	engine := NewMockSombraEngineCase(ctrl)

	patterns := []*entities.Pattern{{Pattern: "/legacy/**", Delete: true}}
	scanner.EXPECT().
		ScanTree("/target", []entities.Wildcard{"**/*"}, nil).
		Return(createScanResultChannel([]entities.FileScanResult{{File: "/legacy/old.go"}}))
	engine.EXPECT().MatchDelete(entities.File("/legacy/old.go"), patterns).Return(true, nil)
	files.EXPECT().Remove("/target", entities.File("/legacy/old.go")).Return(errors.New("permission denied"))

	_, err := pruneTarget(scanner, files, engine, "/target", patterns, true)
	if err == nil {
		t.Fatal("pruneTarget() expected error")
	}
}
