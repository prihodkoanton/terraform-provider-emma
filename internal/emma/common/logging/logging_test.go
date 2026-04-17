package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "sanitize password field",
			input: map[string]interface{}{
				"username": "admin",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"username": "admin",
				"password": "[REDACTED]",
			},
		},
		{
			name: "sanitize token field",
			input: map[string]interface{}{
				"user_id":      "123",
				"access_token": "abc123xyz",
			},
			expected: map[string]interface{}{
				"user_id":      "123",
				"access_token": "[REDACTED]",
			},
		},
		{
			name: "sanitize multiple sensitive fields",
			input: map[string]interface{}{
				"name":          "test",
				"api_key":       "key123",
				"client_secret": "secret456",
				"token":         "token789",
			},
			expected: map[string]interface{}{
				"name":          "test",
				"api_key":       "[REDACTED]",
				"client_secret": "[REDACTED]",
				"token":         "[REDACTED]",
			},
		},
		{
			name: "sanitize nested map",
			input: map[string]interface{}{
				"user": "admin",
				"auth": map[string]interface{}{
					"password": "secret",
					"token":    "abc123",
				},
			},
			expected: map[string]interface{}{
				"user": "admin",
				"auth": map[string]interface{}{
					"password": "[REDACTED]",
					"token":    "[REDACTED]",
				},
			},
		},
		{
			name: "case insensitive field matching",
			input: map[string]interface{}{
				"Password":    "secret",
				"ACCESS_TOKEN": "token",
				"ApiKey":      "key",
			},
			expected: map[string]interface{}{
				"Password":    "[REDACTED]",
				"ACCESS_TOKEN": "[REDACTED]",
				"ApiKey":      "[REDACTED]",
			},
		},
		{
			name: "preserve non-sensitive fields",
			input: map[string]interface{}{
				"name":        "test-resource",
				"description": "test description",
				"count":       5,
				"enabled":     true,
			},
			expected: map[string]interface{}{
				"name":        "test-resource",
				"description": "test description",
				"count":       5,
				"enabled":     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sanitize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{"password field", "password", true},
		{"Password field uppercase", "Password", true},
		{"PASSWORD field all caps", "PASSWORD", true},
		{"user_password field", "user_password", true},
		{"token field", "token", true},
		{"access_token field", "access_token", true},
		{"refresh_token field", "refresh_token", true},
		{"api_key field", "api_key", true},
		{"secret field", "secret", true},
		{"client_secret field", "client_secret", true},
		{"private_key field", "private_key", true},
		{"authorization field", "authorization", true},
		{"bearer field", "bearer", true},
		{"credentials field", "credentials", true},
		{"normal field", "username", false},
		{"name field", "name", false},
		{"description field", "description", false},
		{"id field", "id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveField(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "normal string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer abc123xyz",
			expected: "Authorization: Bearer [REDACTED]",
		},
		{
			name:     "bearer token lowercase",
			input:    "authorization: bearer abc123xyz",
			expected: "authorization: Bearer [REDACTED]",
		},
		{
			name:     "basic auth",
			input:    "Authorization: Basic dXNlcjpwYXNz",
			expected: "Authorization: Basic [REDACTED]",
		},
		{
			name:     "basic auth lowercase",
			input:    "authorization: basic dXNlcjpwYXNz",
			expected: "authorization: Basic [REDACTED]",
		},
		{
			name:     "multiple tokens",
			input:    "Bearer token1 and Basic token2",
			expected: "Bearer [REDACTED] and Basic [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeWithStringValues(t *testing.T) {
	input := map[string]interface{}{
		"name":          "test",
		"authorization": "Bearer abc123xyz",
		"header":        "Basic dXNlcjpwYXNz",
	}

	result := Sanitize(input)

	// The authorization field should be redacted because it's a sensitive field name
	assert.Equal(t, "[REDACTED]", result["authorization"])
	
	// The header field should have its bearer token sanitized
	assert.Equal(t, "Basic [REDACTED]", result["header"])
	
	// The name field should be unchanged
	assert.Equal(t, "test", result["name"])
}
