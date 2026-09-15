package usecases

import (
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v3"
)

func TestSombraEngineMatchHonorsWhen(t *testing.T) {
	ctrl := gomock.NewController(t)
	dirManager := NewMockDirectoryManagerPort(ctrl)
	engine := NewSombraEngineInteractor(dirManager, nil, nil)

	file := entities.File("/.github/workflows/ci.yml")
	disabled := entities.Condition(false)

	// The disabled pattern must never be matched against, so only the enabled
	// pattern registers expectations.
	patterns := []*entities.Pattern{
		{Pattern: "/.github/**", When: &disabled, Content: entities.Mappings{"a": "b"}},
		{Pattern: "/.github/**", Content: entities.Mappings{"c": "d"}},
	}

	dirManager.EXPECT().PathMatch(file, entities.Wildcard("/.github/**")).Return(true, nil)
	for _, ignore := range engine.alwaysIgnore {
		dirManager.EXPECT().PathMatch(file, ignore).Return(false, nil).AnyTimes()
	}

	include, res, err := engine.Match(file, patterns)
	if err != nil {
		t.Fatalf("Match() unexpected error: %v", err)
	}
	if !include {
		t.Fatal("Match() include = false, want true")
	}
	if len(res) != 1 || res[0] != patterns[1] {
		t.Fatalf("Match() returned %#v, want only the enabled pattern", res)
	}
}

func TestSombraEngineMatchDisabledWhenSkipsPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	dirManager := NewMockDirectoryManagerPort(ctrl)
	engine := NewSombraEngineInteractor(dirManager, nil, nil)

	disabled := entities.Condition(false)
	patterns := []*entities.Pattern{
		{Pattern: "/**/*", When: &disabled, Content: entities.Mappings{"a": "b"}},
	}

	include, res, err := engine.Match(entities.File("/src/main.go"), patterns)
	if err != nil {
		t.Fatalf("Match() unexpected error: %v", err)
	}
	if include {
		t.Fatal("Match() include = true, want false")
	}
	if len(res) != 0 {
		t.Fatalf("Match() returned %d patterns, want 0", len(res))
	}
}

func TestPatternWhenDecodesRenderedAndUnrendered(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		enabled bool
	}{
		{
			name:    "rendered true",
			yaml:    "pattern: /a\nwhen: true\n",
			enabled: true,
		},
		{
			name:    "rendered false",
			yaml:    "pattern: /a\nwhen: false\n",
			enabled: false,
		},
		{
			name:    "unrendered template stays enabled",
			yaml:    "pattern: /a\nwhen: '{{ eq .include_ci \"true\" }}'\n",
			enabled: true,
		},
		{
			name:    "absent stays enabled",
			yaml:    "pattern: /a\n",
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pattern entities.Pattern
			if err := yaml.Unmarshal([]byte(tt.yaml), &pattern); err != nil {
				t.Fatalf("yaml.Unmarshal() unexpected error: %v", err)
			}
			if got := patternEnabled(&pattern); got != tt.enabled {
				t.Fatalf("patternEnabled() = %v, want %v", got, tt.enabled)
			}
		})
	}
}

func TestSombraEngineMatchDisabledWhenSkipsTombstone(t *testing.T) {
	ctrl := gomock.NewController(t)
	dirManager := NewMockDirectoryManagerPort(ctrl)
	engine := NewSombraEngineInteractor(dirManager, nil, nil)

	disabled := entities.Condition(false)
	patterns := []*entities.Pattern{
		{Pattern: "/legacy/**", Delete: true, When: &disabled},
	}

	match, err := engine.MatchDelete(entities.File("/legacy/old.go"), patterns)
	if err != nil {
		t.Fatalf("MatchDelete() unexpected error: %v", err)
	}
	if match {
		t.Fatal("MatchDelete() = true, want false")
	}
}
