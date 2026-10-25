package usecases

type RepositoryPrepareCase interface {
	Prepare(uri, tag string) (RepositoryPort, error)
}

type RepositoryPrepareInteractor struct {
	factory RepositoryFactory
}

func (l *RepositoryPrepareInteractor) Prepare(uri, tag string) (RepositoryPort, error) {
	repo, err := l.factory(uri)
	if err != nil {
		return nil, err
	}

	err = repo.Clone()
	if err != nil {
		return nil, err
	}

	if tag == "" {
		return repo, nil
	}

	_, err = repo.Use(tag)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func NewRepositoryPrepareInteractor(factory RepositoryFactory) *RepositoryPrepareInteractor {
	return &RepositoryPrepareInteractor{factory: factory}
}

var _ RepositoryPrepareCase = (*RepositoryPrepareInteractor)(nil)
