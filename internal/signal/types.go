package signal

import "fmt"

// MessageResponse represents the result of receiving a Signal message
type MessageResponse struct {
	// Message is the text content of the received message
	Message string `json:"message,omitempty"`

	// SenderID is the phone number of the message sender (e.g., "+15551234567")
	SenderID string `json:"sender_id,omitempty"`

	// GroupName is the name of the group if the message was sent to a group
	GroupName string `json:"group_name,omitempty"`

	// Error contains any error message if the receive operation failed
	Error string `json:"error,omitempty"`
}

// SignalError represents an error that occurred during Signal operations
type SignalError struct {
	// Op is the operation that failed (e.g., "send", "receive", "listGroups")
	Op string

	// UserID is the phone number associated with the operation
	UserID string

	// Err is the underlying error
	Err error
}

// Error implements the error interface
func (e *SignalError) Error() string {
	if e.UserID != "" {
		return fmt.Sprintf("signal %s (user: %s): %v", e.Op, e.UserID, e.Err)
	}
	return fmt.Sprintf("signal %s: %v", e.Op, e.Err)
}

// Unwrap implements the errors.Unwrap interface
func (e *SignalError) Unwrap() error {
	return e.Err
}

// Group represents a Signal group
type Group struct {
	// ID is the internal group identifier
	ID string

	// Name is the human-readable group name
	Name string

	// Members is a list of phone numbers in the group
	Members []string
}
