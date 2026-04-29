package logging

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SensitiveFields contains patterns for fields that should be sanitized
var SensitiveFields = []string{
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
}

// Sanitize removes sensitive data from a map of fields
// It replaces sensitive values with "[REDACTED]"
func Sanitize(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	for key, value := range fields {
		if isSensitiveField(key) {
			sanitized[key] = "[REDACTED]"
		} else {
			// Handle nested maps
			if nestedMap, ok := value.(map[string]interface{}); ok {
				sanitized[key] = Sanitize(nestedMap)
			} else if strValue, ok := value.(string); ok {
				// Sanitize string values that might contain sensitive data
				sanitized[key] = sanitizeString(strValue)
			} else {
				sanitized[key] = value
			}
		}
	}
	return sanitized
}

// isSensitiveField checks if a field name indicates sensitive data
func isSensitiveField(fieldName string) bool {
	lowerField := strings.ToLower(fieldName)
	for _, sensitive := range SensitiveFields {
		if strings.Contains(lowerField, sensitive) {
			return true
		}
	}
	return false
}

// sanitizeString removes sensitive patterns from string values
func sanitizeString(value string) string {
	// Redact bearer tokens - must have at least 2 non-= characters
	bearerPattern := regexp.MustCompile(`(?i)bearer\s+([a-zA-Z0-9\-._~+/=]+)`)
	value = bearerPattern.ReplaceAllStringFunc(value, func(match string) string {
		// Extract the token part
		parts := regexp.MustCompile(`(?i)bearer\s+`).Split(match, 2)
		if len(parts) == 2 {
			token := parts[1]
			// Only redact if token has at least 2 non-= characters
			nonEquals := strings.Trim(token, "=")
			if len(nonEquals) >= 2 {
				return "Bearer [REDACTED]"
			}
		}
		return match
	})

	// Redact basic auth - must have at least 2 non-= characters
	basicAuthPattern := regexp.MustCompile(`(?i)basic\s+([a-zA-Z0-9+/=]+)`)
	value = basicAuthPattern.ReplaceAllStringFunc(value, func(match string) string {
		// Extract the credentials part
		parts := regexp.MustCompile(`(?i)basic\s+`).Split(match, 2)
		if len(parts) == 2 {
			creds := parts[1]
			// Only redact if credentials have at least 2 non-= characters
			nonEquals := strings.Trim(creds, "=")
			if len(nonEquals) >= 2 {
				return "Basic [REDACTED]"
			}
		}
		return match
	})

	return value
}

// LogAPIRequest logs an API request at debug level with sanitized data
func LogAPIRequest(ctx context.Context, method, path string, body interface{}) {
	fields := map[string]interface{}{
		"method": method,
		"path":   path,
	}

	if body != nil {
		if bodyMap, ok := body.(map[string]interface{}); ok {
			fields["body"] = Sanitize(bodyMap)
		} else {
			fields["body"] = body
		}
	}

	tflog.Debug(ctx, "API Request", fields)
}

// LogAPIResponse logs an API response at debug level with sanitized data
func LogAPIResponse(ctx context.Context, statusCode int, body interface{}) {
	fields := map[string]interface{}{
		"status_code": statusCode,
	}

	if body != nil {
		if bodyMap, ok := body.(map[string]interface{}); ok {
			fields["body"] = Sanitize(bodyMap)
		} else {
			fields["body"] = body
		}
	}

	tflog.Debug(ctx, "API Response", fields)
}

// LogStateTransition logs a state transition at info level
func LogStateTransition(ctx context.Context, resourceType, resourceID, fromState, toState string) {
	tflog.Info(ctx, "State transition", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"from_state":    fromState,
		"to_state":      toState,
	})
}

// LogResourceCreated logs successful resource creation at info level
func LogResourceCreated(ctx context.Context, resourceType, resourceID string) {
	tflog.Info(ctx, "Resource created", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

// LogResourceUpdated logs successful resource update at info level
func LogResourceUpdated(ctx context.Context, resourceType, resourceID string) {
	tflog.Info(ctx, "Resource updated", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

// LogResourceDeleted logs successful resource deletion at info level
func LogResourceDeleted(ctx context.Context, resourceType, resourceID string) {
	tflog.Info(ctx, "Resource deleted", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

// LogResourceRead logs successful resource read at info level
func LogResourceRead(ctx context.Context, resourceType, resourceID string) {
	tflog.Info(ctx, "Resource read", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

// LogError logs an error with full context at error level
func LogError(ctx context.Context, operation, resourceType, resourceID string, err error) {
	fields := map[string]interface{}{
		"operation":     operation,
		"resource_type": resourceType,
		"error":         err.Error(),
	}

	if resourceID != "" {
		fields["resource_id"] = resourceID
	}

	tflog.Error(ctx, fmt.Sprintf("%s failed", operation), fields)
}

// LogRetryAttempt logs a retry attempt at debug level
func LogRetryAttempt(ctx context.Context, operation string, attempt, maxAttempts int, delay string) {
	tflog.Debug(ctx, "Retry attempt", map[string]interface{}{
		"operation":    operation,
		"attempt":      attempt,
		"max_attempts": maxAttempts,
		"delay":        delay,
	})
}

// LogAsyncOperation logs async operation status at debug level
func LogAsyncOperation(ctx context.Context, resourceType, resourceID, status string) {
	tflog.Debug(ctx, "Async operation status", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"status":        status,
	})
}
