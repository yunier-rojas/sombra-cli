package analysers

import (
	"fmt"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"sort"
	"sync"
)

type AnalyserFactory func(baseDir, fn string) (usecases.LocalFileAnalyserPort, error)
type ApplyFunc func(baseDir, fn string) bool

type entry struct {
	factory AnalyserFactory
	apply   ApplyFunc
	weight  int
}

type Registry struct {
	mu      sync.RWMutex
	entries []entry
}

func (r *Registry) GetAnalyser(baseDir string, file entities.File) (usecases.LocalFileAnalyserPort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn := string(file)
	for _, e := range r.entries {
		if e.apply(baseDir, fn) {
			logger.Info(fmt.Sprintf("Analyser found for %s", fn))
			return e.factory(baseDir, fn)
		}
	}
	err := fmt.Errorf("no analyser found for %s", fn)
	logger.Error("Failed to get analyser", err)
	return nil, err
}

var globalRegistry = &Registry{
	entries: make([]entry, 0),
}

func GetRegistry() usecases.FileAnalyserRegistryPort {
	return globalRegistry
}

func Register(apply ApplyFunc, factory AnalyserFactory, weight int) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.entries = append(globalRegistry.entries, entry{apply: apply, factory: factory, weight: weight})
	sort.SliceStable(globalRegistry.entries, func(i, j int) bool {
		return globalRegistry.entries[i].weight < globalRegistry.entries[j].weight
	})
}

var _ usecases.FileAnalyserRegistryPort = (*Registry)(nil)
