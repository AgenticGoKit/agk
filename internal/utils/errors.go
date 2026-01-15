package utils

import (
	"fmt"
)

// UserError represents an error with a user-friendly message and solution
type UserError struct {
	Message  string
	Solution string
	Err      error
}

func (e *UserError) Error() string {
	msg := e.Message
	if e.Solution != "" {
		msg += fmt.Sprintf("\n\n💡 Solution: %s", e.Solution)
	}
	if e.Err != nil {
		msg += fmt.Sprintf("\n\nDetails: %v", e.Err)
	}
	return msg
}

func (e *UserError) Unwrap() error {
	return e.Err
}

// NewUserError creates a new UserError
func NewUserError(message, solution string, err error) *UserError {
	return &UserError{
		Message:  message,
		Solution: solution,
		Err:      err,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
