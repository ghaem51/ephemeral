package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrValidation        = errors.New("validation failed")
)

type TransitionError struct {
	Entity string
	From   string
	To     string
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("%s: %s cannot transition from %s to %s", ErrInvalidTransition, e.Entity, e.From, e.To)
}

func (e *TransitionError) Unwrap() error {
	return ErrInvalidTransition
}
