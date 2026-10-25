package usecases

type RepositoryLifecyclePort interface {
	Clone() error
	Clean() error
}

type RepositoryCheckoutPort interface {
	Dir() string
	Use(version string) (string, error)
	GetTags() ([]string, error)
}

type RepositoryPort interface {
	RepositoryLifecyclePort
	RepositoryCheckoutPort
}

type RepositoryFactory func(uri string) (RepositoryPort, error)
