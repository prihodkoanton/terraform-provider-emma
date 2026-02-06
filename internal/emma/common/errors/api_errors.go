package errors

import (
	"fmt"
	"net/http"
)

// MapHTTPError maps HTTP status codes to user-friendly messages
func MapHTTPError(statusCode int, apiMessage string) string {
	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Sprintf("Invalid request: %s", apiMessage)
	case http.StatusUnauthorized:
		return "Authentication failed. Please check your credentials."
	case http.StatusForbidden:
		return "Permission denied. You don't have access to this resource."
	case http.StatusNotFound:
		return "Resource not found. It may have been deleted."
	case http.StatusConflict:
		return fmt.Sprintf("Resource conflict: %s", apiMessage)
	case http.StatusUnprocessableEntity:
		return fmt.Sprintf("Validation error: %s", apiMessage)
	case http.StatusTooManyRequests:
		return "Rate limit exceeded. Please try again later."
	case http.StatusInternalServerError:
		return "Server error. Please try again or contact support."
	case http.StatusServiceUnavailable:
		return "Service temporarily unavailable. Please try again later."
	default:
		return fmt.Sprintf("API error (status %d): %s", statusCode, apiMessage)
	}
}

// IsRetryable determines if an error should be retried based on HTTP status code
func IsRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode >= 500
}
