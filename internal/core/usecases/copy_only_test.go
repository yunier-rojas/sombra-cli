package usecases

import (
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestSombraEngineCombineCopyOnlyKeepsContentMappings(t *testing.T) {
	engine := NewSombraEngineInteractor(nil, nil, nil)

	res := engine.Combine([]*entities.Pattern{
		{
			Pattern:  "/go.mod",
			CopyOnly: true,
			Default:  entities.Mappings{"name": "acme"},
			Content:  entities.Mappings{"word:sombra": "app"},
		},
	})

	if len(res.Content) == 0 {
		t.Fatal("Combine() must keep Content mappings for copy_only patterns")
	}
	if len(res.Path) == 0 || len(res.Name) == 0 {
		t.Fatalf("Combine() must keep path/name mappings, got %v / %v", res.Path, res.Name)
	}
}

func TestWithoutCopyOnlyDropsCopyOnlyPatterns(t *testing.T) {
	normal := &entities.Pattern{Pattern: "/cmd/**"}
	copyOnly := &entities.Pattern{Pattern: "/go.mod", CopyOnly: true}
	tombstone := &entities.Pattern{Pattern: "/legacy/**", Delete: true}

	filtered := withoutCopyOnly([]*entities.Pattern{normal, copyOnly, tombstone})

	if len(filtered) != 2 {
		t.Fatalf("withoutCopyOnly() kept %d patterns, want 2", len(filtered))
	}
	for _, pattern := range filtered {
		if pattern.CopyOnly {
			t.Fatalf("withoutCopyOnly() kept copy_only pattern %q", pattern.Pattern)
		}
	}
}
