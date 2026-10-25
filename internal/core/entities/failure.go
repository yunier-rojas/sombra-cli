package entities

import "strings"

// FailureKind classifies a domain failure without exposing the platform type
// that caused it.
type FailureKind string

const (
	FailureNotFound    FailureKind = "not-found"
	FailurePermission  FailureKind = "permission"
	FailureInvalid     FailureKind = "invalid"
	FailureUnavailable FailureKind = "unavailable"
	FailureStorage     FailureKind = "storage"
)

var (
	ErrNotFound    = &Failure{Kind: FailureNotFound}
	ErrPermission  = &Failure{Kind: FailurePermission}
	ErrInvalid     = &Failure{Kind: FailureInvalid}
	ErrUnavailable = &Failure{Kind: FailureUnavailable}
	ErrStorage     = &Failure{Kind: FailureStorage}
)

// Failure is a domain failure with an operation and an optional cause.
type Failure struct {
	Kind FailureKind
	Op   string
	Err  error
}

func (f *Failure) Error() string {
	if f == nil {
		return ""
	}

	parts := make([]string, 0, 3)
	if f.Op != "" {
		parts = append(parts, f.Op)
	}
	parts = append(parts, string(f.Kind))
	if f.Err != nil {
		parts = append(parts, f.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Err
}

// Is matches failures by kind so errors.Is works with the sentinels.
func (f *Failure) Is(target error) bool {
	other, ok := target.(*Failure)
	return ok && other != nil && f != nil && f.Kind == other.Kind
}

// NewFailure builds a classified failure.
func NewFailure(kind FailureKind, op string, err error) *Failure {
	return &Failure{Kind: kind, Op: op, Err: err}
}

func NotFound(op string, err error) *Failure {
	return NewFailure(FailureNotFound, op, err)
}

func Permission(op string, err error) *Failure {
	return NewFailure(FailurePermission, op, err)
}

func Invalid(op string, err error) *Failure {
	return NewFailure(FailureInvalid, op, err)
}

func Unavailable(op string, err error) *Failure {
	return NewFailure(FailureUnavailable, op, err)
}

func Storage(op string, err error) *Failure {
	return NewFailure(FailureStorage, op, err)
}
