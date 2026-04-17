package logging

import (
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: provider-improvements, Property 11: Sensitive Data Not Logged
// Validates: Requirements 10.4, 14.2
//
// Property: For any logging operation, sensitive fields (passwords, tokens, keys)
// should be redacted or omitted.
func TestProperty_SensitiveDataNotLogged(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Sensitive field values are always redacted
	properties.Property("sensitive field values are redacted", prop.ForAll(
		func(fieldName string, fieldValue string) bool {
			// Create a map with the generated field
			fields := map[string]interface{}{
				fieldName: fieldValue,
			}

			// Sanitize the fields
			sanitized := Sanitize(fields)

			// If the field name is sensitive, the value should be redacted
			if isSensitiveField(fieldName) {
				return sanitized[fieldName] == "[REDACTED]"
			}

			// If the field name is not sensitive, the value should be preserved
			return sanitized[fieldName] == fieldValue
		},
		genSensitiveFieldName(),
		gen.AnyString(),
	))

	// Property: Nested sensitive fields are redacted
	properties.Property("nested sensitive fields are redacted", prop.ForAll(
		func(outerKey string, innerKey string, innerValue string) bool {
			// Create a nested map
			fields := map[string]interface{}{
				outerKey: map[string]interface{}{
					innerKey: innerValue,
				},
			}

			// Sanitize the fields
			sanitized := Sanitize(fields)

			// If the outer key is sensitive, the entire value should be redacted
			if isSensitiveField(outerKey) {
				return sanitized[outerKey] == "[REDACTED]"
			}

			// If the outer key is not sensitive, check the nested map
			nestedMap, ok := sanitized[outerKey].(map[string]interface{})
			if !ok {
				return false
			}

			// If the inner key is sensitive, the value should be redacted
			if isSensitiveField(innerKey) {
				return nestedMap[innerKey] == "[REDACTED]"
			}

			// If the inner key is not sensitive, the value should be preserved
			return nestedMap[innerKey] == innerValue
		},
		gen.Identifier(),
		genSensitiveFieldName(),
		gen.AnyString(),
	))

	// Property: Bearer tokens in strings are redacted
	properties.Property("bearer tokens in strings are redacted", prop.ForAll(
		func(prefix string, token string, suffix string) bool {
			// Skip if token has less than 2 non-= characters
			nonEquals := strings.Trim(token, "=")
			if len(nonEquals) < 2 {
				return true // Skip this test case
			}

			// Create a string with a bearer token
			input := prefix + "Bearer " + token + suffix

			// Sanitize the string
			result := sanitizeString(input)

			// The result should not contain the original token
			return !strings.Contains(result, token)
		},
		gen.AlphaString(),
		gen.RegexMatch("[a-zA-Z0-9._~+/=-]{2,}"), // Valid token characters, at least 2 chars
		gen.AlphaString(),
	))

	// Property: Basic auth in strings is redacted
	properties.Property("basic auth in strings are redacted", prop.ForAll(
		func(prefix string, credentials string, suffix string) bool {
			// Skip if credentials have less than 2 non-= characters
			nonEquals := strings.Trim(credentials, "=")
			if len(nonEquals) < 2 {
				return true // Skip this test case
			}

			// Create a string with basic auth
			input := prefix + "Basic " + credentials + suffix

			// Sanitize the string
			result := sanitizeString(input)

			// The result should not contain the original credentials
			return !strings.Contains(result, credentials)
		},
		gen.AlphaString(),
		gen.RegexMatch("[a-zA-Z0-9+/=]{2,}"), // Valid base64 characters, at least 2 chars
		gen.AlphaString(),
	))

	// Property: Non-sensitive fields are preserved
	properties.Property("non-sensitive fields are preserved", prop.ForAll(
		func(fields map[string]string) bool {
			// Convert to map[string]interface{}
			input := make(map[string]interface{})
			for k, v := range fields {
				input[k] = v
			}

			// Sanitize
			sanitized := Sanitize(input)

			// Check that all non-sensitive fields are preserved
			for key, value := range fields {
				if !isSensitiveField(key) {
					if sanitized[key] != value {
						return false
					}
				}
			}

			return true
		},
		genNonSensitiveFields(),
	))

	// Property: Sanitize is idempotent
	properties.Property("sanitize is idempotent", prop.ForAll(
		func(fields map[string]string) bool {
			// Convert to map[string]interface{}
			input := make(map[string]interface{})
			for k, v := range fields {
				input[k] = v
			}

			// Sanitize once
			sanitized1 := Sanitize(input)

			// Sanitize again
			sanitized2 := Sanitize(sanitized1)

			// Results should be equal
			if len(sanitized1) != len(sanitized2) {
				return false
			}

			for key := range sanitized1 {
				if sanitized1[key] != sanitized2[key] {
					return false
				}
			}

			return true
		},
		gen.MapOf(gen.Identifier(), gen.AnyString()),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genSensitiveFieldName generates field names that may or may not be sensitive
func genSensitiveFieldName() gopter.Gen {
	return gen.OneGenOf(
		// Sensitive field names
		gen.OneConstOf(
			"password",
			"token",
			"secret",
			"key",
			"access_token",
			"refresh_token",
			"api_key",
			"private_key",
			"client_secret",
			"authorization",
			"bearer",
			"credentials",
			"Password",
			"TOKEN",
			"user_password",
			"api_secret",
		),
		// Non-sensitive field names
		gen.OneConstOf(
			"name",
			"id",
			"description",
			"username",
			"email",
			"status",
			"type",
			"count",
			"enabled",
		),
	)
}

// genNonSensitiveFields generates a map of non-sensitive fields
func genNonSensitiveFields() gopter.Gen {
	nonSensitiveKeys := gen.OneConstOf(
		"name",
		"id",
		"description",
		"username",
		"email",
		"status",
		"type",
		"count",
		"enabled",
		"resource_type",
		"operation",
		"method",
		"path",
	)

	return gen.MapOf(nonSensitiveKeys, gen.AnyString())
}
