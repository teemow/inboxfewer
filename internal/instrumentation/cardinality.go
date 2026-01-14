package instrumentation

import (
	"github.com/teemow/inboxfewer/internal/logging"
)

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
// Returns "unknown" for invalid or empty emails to ensure metric labels always have a value.
//
// Example:
//
//	ExtractUserDomain("jane@example.com")  // "example.com"
//	ExtractUserDomain("user@gmail.com")    // "gmail.com"
//	ExtractUserDomain("invalid")           // "unknown"
//	ExtractUserDomain("")                  // "unknown"
func ExtractUserDomain(email string) string {
	domain := logging.ExtractDomain(email)
	if domain == "" {
		return "unknown"
	}
	return domain
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
