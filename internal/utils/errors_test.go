package utils

import (
	"errors"
	"testing"
)

func TestUserError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		solution string
		err      error
		want     string
	}{
		{
			name:     "with solution and error",
			message:  "Failed to load config",
			solution: "Check if config file exists",
			err:      errors.New("file not found"),
			want:     "Failed to load config\n\n💡 Solution: Check if config file exists\n\nDetails: file not found",
		},
		{
			name:     "without solution",
			message:  "Invalid input",
			solution: "",
			err:      nil,
			want:     "Invalid input",
		},
		{
			name:     "with solution only",
			message:  "Failed to create file",
			solution: "Check file permissions",
			err:      nil,
			want:     "Failed to create file\n\n💡 Solution: Check file permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ue := NewUserError(tt.message, tt.solution, tt.err)
			if got := ue.Error(); got != tt.want {
				t.Errorf("UserError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	ve := NewValidationError("name", "cannot be empty")
	want := "name: cannot be empty"

	if got := ve.Error(); got != want {
		t.Errorf("ValidationError.Error() = %v, want %v", got, want)
	}
}

func TestUserErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	ue := NewUserError("wrapper", "solution", originalErr)

	if err := ue.Unwrap(); !errors.Is(err, originalErr) {
		t.Error("Unwrap() did not return original error")
	}
}
