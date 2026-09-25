package usecases

type CliVersionCase interface {
	GetVersion() string
}

type CliVersionInteractor struct {
	version string
}

func (v *CliVersionInteractor) GetVersion() string {
	return v.version
}

func NewCliVersionInteractor(version string) *CliVersionInteractor {
	return &CliVersionInteractor{version: version}
}

var _ CliVersionCase = (*CliVersionInteractor)(nil)
