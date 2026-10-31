package domain

import (
	"errors"
	"testing"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"
)

func TestExample_CreateThenRename(t *testing.T) {
	e := New("id-1")

	if err := e.Create("first"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if e.Name() != "first" {
		t.Fatalf("name = %q, want first", e.Name())
	}
	// Two uncommitted? No — only ExampleCreated so far.
	if got := len(e.Uncommitted()); got != 1 {
		t.Fatalf("uncommitted = %d, want 1", got)
	}

	if err := e.Rename("second"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if e.Name() != "second" {
		t.Fatalf("name = %q, want second", e.Name())
	}
}

func TestExample_CreateTwice_IsValidationError(t *testing.T) {
	e := New("id-2")
	if err := e.Create("a"); err != nil {
		t.Fatalf("create: %v", err)
	}
	err := e.Create("b")
	if !errors.Is(err, cqrs.ErrValidation) {
		t.Fatalf("second create err = %v, want ErrValidation", err)
	}
}

func TestExample_RenameBeforeCreate_IsValidationError(t *testing.T) {
	e := New("id-3")
	err := e.Rename("x")
	if !errors.Is(err, cqrs.ErrValidation) {
		t.Fatalf("rename err = %v, want ErrValidation", err)
	}
}
