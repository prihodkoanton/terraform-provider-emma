package retry

import (
	"net/http"
	"strings"
	"time"
)

// IsStateConflictError checks if error is due to resource state conflict
func IsStateConflictError(err error, statusCode int, apiError string) bool {
	if statusCode == http.StatusConflict {
		return true
	}

	// Check for state-related error messages
	stateErrorKeywords := []string{
		"state",
		"busy",
		"recomposing",
		"inappropriate compute instance state",
		"resource conflict",
	}

	lowerError := strings.ToLower(apiError)
	for _, keyword := range stateErrorKeywords {
		if strings.Contains(lowerError, keyword) {
			return true
		}
	}

	return false
}

// StateConflictRetryConfig creates retry config for state conflicts
func StateConflictRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 2 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		ShouldRetry: func(err error) bool {
			// This will be set by the caller with actual error context
			return true
		},
	}
}
