// Package logging provides structured logging utilities for the inboxfewer application.
//
// This package centralizes logging patterns to ensure consistent, structured logging
// throughout the codebase using the standard library's slog package.
//
// # Key Features
//
//   - Structured logging with slog
//   - PII sanitization (email anonymization)
//   - Consistent attribute naming across the codebase
//   - Logger adapter interface for flexibility
//
// # Usage Patterns
//
// Create a logger with standard attributes:
//
//	logger := logging.WithOperation(slog.Default(), "gmail.list")
//	logger.Info("listing emails",
//	    logging.Status("success"))
//
// Sanitize sensitive data before logging:
//
//	logger.Info("user operation",
//	    logging.UserHash(email))
//
// # Security Considerations
//
// This package is designed with security in mind:
//   - User emails are hashed to prevent PII leakage while allowing correlation
//   - Tokens are never logged directly
package logging
