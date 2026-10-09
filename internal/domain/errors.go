package domain

import "errors"

// Sentinel errors. Callers use errors.Is to check them.
// This is the idiomatic Go error pattern — stable, comparable, greppable.
var (
	ErrBookNotFound      = errors.New("book not found")
	ErrInvalidTitle      = errors.New("title is required")
	ErrInvalidAuthor     = errors.New("author is required")
	ErrInvalidTotalPages = errors.New("total pages must be greater than 0")
	ErrInvalidProgress   = errors.New("pages read must be between 0 and total pages")
	ErrInvalidStatus     = errors.New("invalid status")
)
