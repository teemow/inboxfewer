package instrumentation

import "strings"

// Cardinality management helpers for metrics.
// These functions reduce high-cardinality label values to prevent metrics explosion.
//
// # Warning
//
// High cardinality in metrics can cause:
// - Increased memory usage in Prometheus/metrics backends
// - Slower query performance
// - Higher storage costs
//
// Always use these helpers when recording metrics with user identifiers.

// ExtractUserDomain extracts the domain part from an email address.
// This reduces cardinality by using the domain instead of the full email.
//
// Example:
//
//	ExtractUserDomain("jane@example.com")  // "example.com"
//	ExtractUserDomain("user@gmail.com")    // "gmail.com"
//	ExtractUserDomain("invalid")           // "unknown"
//	ExtractUserDomain("")                  // "unknown"
func ExtractUserDomain(email string) string {
	if email == "" {
		return "unknown"
	}

	parts := strings.Split(email, "@")
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}

	return "unknown"
}

// Common operation types for Google API metrics.
// Status, OAuth, and Service constants are defined in config.go.
const (
	OperationList   = "list"
	OperationGet    = "get"
	OperationCreate = "create"
	OperationUpdate = "update"
	OperationDelete = "delete"
	OperationSend   = "send"
	OperationSearch = "search"
)
