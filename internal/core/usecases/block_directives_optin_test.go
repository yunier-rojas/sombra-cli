package usecases

import (
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestSombraEngineCombineBlockDirectivesOptIn(t *testing.T) {
	engine := NewSombraEngineInteractor(nil, nil, nil)

	if got := engine.Combine([]*entities.Pattern{{Pattern: "/a"}}); got.BlockDirectives {
		t.Fatal("Combine() BlockDirectives = true, want false by default")
	}

	got := engine.Combine([]*entities.Pattern{
		{Pattern: "/a"},
		{Pattern: "/b", BlockDirectives: true},
	})
	if !got.BlockDirectives {
		t.Fatal("Combine() BlockDirectives = false, want true when a pattern opts in")
	}
}
