package entities

import (
	"errors"
	"testing"
)

func TestFailureMatchesByKind(t *testing.T) {
	cause := errors.New("permission denied")
	err := Permission("read file", cause)

	if !errors.Is(err, ErrPermission) {
		t.Fatal("expected ErrPermission")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("unexpected ErrNotFound")
	}
	if !errors.Is(err, cause) {
		t.Fatal("cause was not preserved")
	}
	if err.Error() != "read file: permission: permission denied" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestFailureSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"not found", NotFound("load", nil), ErrNotFound},
		{"permission", Permission("read", nil), ErrPermission},
		{"invalid", Invalid("parse", nil), ErrInvalid},
		{"unavailable", Unavailable("clone", nil), ErrUnavailable},
		{"storage", Storage("write", nil), ErrStorage},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("%v does not match %v", tt.err, tt.want)
			}
		})
	}
}
