package logging

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
)

// Common log attribute keys for consistent naming across the codebase.
const (
	KeyOperation = "operation"
	KeyService   = "service"
	KeyAccount   = "account"
	KeyUserHash  = "user_hash"
	KeyDuration  = "duration"
	KeyStatus    = "status"
	KeyError     = "error"
	KeyTool      = "tool"
)

// Status values for consistent logging.
// Note: These are intentionally duplicated from instrumentation package
// to avoid circular dependencies (instrumentation imports logging).
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

// WithOperation returns a logger with the operation attribute set.
func WithOperation(logger *slog.Logger, operation string) *slog.Logger {
	return logger.With(slog.String(KeyOperation, operation))
}

// WithTool returns a logger with the tool attribute set.
func WithTool(logger *slog.Logger, tool string) *slog.Logger {
	return logger.With(slog.String(KeyTool, tool))
}

// WithService returns a logger with the service attribute set.
func WithService(logger *slog.Logger, service string) *slog.Logger {
	return logger.With(slog.String(KeyService, service))
}

// WithAccount returns a logger with the account attribute set.
func WithAccount(logger *slog.Logger, account string) *slog.Logger {
	return logger.With(slog.String(KeyAccount, account))
}

// Operation returns a slog attribute for the operation name.
func Operation(op string) slog.Attr {
	return slog.String(KeyOperation, op)
}

// Service returns a slog attribute for the service name.
func Service(svc string) slog.Attr {
	return slog.String(KeyService, svc)
}

// Account returns a slog attribute for the account name.
func Account(account string) slog.Attr {
	return slog.String(KeyAccount, account)
}

// Tool returns a slog attribute for the tool name.
func Tool(tool string) slog.Attr {
	return slog.String(KeyTool, tool)
}

// Status returns a slog attribute for the status.
func Status(status string) slog.Attr {
	return slog.String(KeyStatus, status)
}

// Err returns a slog attribute for an error.
// If err is nil, returns an empty Group attribute that will be omitted from output.
// This allows safely passing Err(maybeNilErr) without adding empty attributes.
//
// Usage:
//
//	logger.Info("operation", logging.Err(err))  // Safe even if err is nil
func Err(err error) slog.Attr {
	if err == nil {
		// Return an empty Group that slog will omit from output
		return slog.Group("")
	}
	return slog.String(KeyError, err.Error())
}

// AnonymizeEmail returns a hashed representation of an email for logging purposes.
// This allows correlation of log entries without exposing PII.
func AnonymizeEmail(email string) string {
	if email == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(email))
	return "user:" + hex.EncodeToString(hash[:8])
}

// UserHash returns a slog attribute with the anonymized user email.
// This is a convenience function to reduce repetition in logging calls and ensure
// consistent attribute naming across the codebase.
//
// Usage:
//
//	logger.Info("operation completed", logging.UserHash(user.Email))
func UserHash(email string) slog.Attr {
	return slog.String(KeyUserHash, AnonymizeEmail(email))
}

// SanitizeToken returns a masked version of a token for logging.
// It returns a length indicator without exposing any token content,
// as even partial token prefixes (like JWT headers) can aid attacks.
func SanitizeToken(token string) string {
	if token == "" {
		return "<empty>"
	}
	return fmt.Sprintf("[token:%d chars]", len(token))
}

// ExtractDomain extracts the domain part from an email address.
// This is useful for lower-cardinality logging where the full email would
// create too many unique values.
func ExtractDomain(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// Domain returns a slog attribute for the email domain (lower cardinality than full email).
func Domain(email string) slog.Attr {
	return slog.String("user_domain", ExtractDomain(email))
}
