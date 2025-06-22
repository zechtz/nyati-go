package api

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "valid email",
			email: "user@example.com",
			want:  true,
		},
		{
			name:  "valid email with subdomain",
			email: "user@mail.example.com",
			want:  true,
		},
		{
			name:  "valid email with numbers",
			email: "user123@example123.com",
			want:  true,
		},
		{
			name:  "valid email with plus",
			email: "user+tag@example.com",
			want:  true,
		},
		{
			name:  "valid email with dash",
			email: "user-name@example.com",
			want:  true,
		},
		{
			name:  "invalid email - no @",
			email: "userexample.com",
			want:  false,
		},
		{
			name:  "invalid email - no domain",
			email: "user@",
			want:  false,
		},
		{
			name:  "invalid email - no user",
			email: "@example.com",
			want:  false,
		},
		{
			name:  "invalid email - no TLD",
			email: "user@example",
			want:  false,
		},
		{
			name:  "invalid email - multiple @",
			email: "user@@example.com",
			want:  false,
		},
		{
			name:  "empty email",
			email: "",
			want:  false,
		},
		{
			name:  "invalid email - spaces",
			email: "user @example.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateEmail(tt.email); got != tt.want {
				t.Errorf("validateEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErrs int
	}{
		{
			name:     "valid strong password",
			password: "StrongPass123!",
			wantErrs: 0,
		},
		{
			name:     "valid password with symbols",
			password: "MyP@ssw0rd!",
			wantErrs: 0,
		},
		{
			name:     "too short",
			password: "Abc1!",
			wantErrs: 1,
		},
		{
			name:     "no uppercase",
			password: "password123!",
			wantErrs: 1,
		},
		{
			name:     "no lowercase",
			password: "PASSWORD123!",
			wantErrs: 1,
		},
		{
			name:     "no numbers",
			password: "Password!",
			wantErrs: 1,
		},
		{
			name:     "no special characters",
			password: "Password123",
			wantErrs: 1,
		},
		{
			name:     "multiple issues",
			password: "pass",
			wantErrs: 4, // short, no upper, no number, no special
		},
		{
			name:     "empty password",
			password: "",
			wantErrs: 5, // all requirements fail (length + 4 character types)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validatePassword(tt.password)
			if len(errors) != tt.wantErrs {
				t.Errorf("validatePassword() errors = %d, want %d. Errors: %v", len(errors), tt.wantErrs, errors)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "text with null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "text with control characters",
			input:    "Hello\x01\x02World\x7f",
			expected: "HelloWorld",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: "",
		},
		{
			name:     "mixed control and valid chars",
			input:    "  Hello\x00\x01World\x7f  ",
			expected: "HelloWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeInput(tt.input); got != tt.expected {
				t.Errorf("sanitizeInput() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestHandleRegister(t *testing.T) {
	// Skip these tests for now since they require a more complex database setup
	t.Skip("Database integration tests require proper migration setup")
}

func TestSanitizeInputInRegister(t *testing.T) {
	// Skip this test for now since it requires database setup
	t.Skip("Database integration tests require proper migration setup")
}