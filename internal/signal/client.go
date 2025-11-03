package signal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Client provides access to Signal messaging operations via signal-cli
type Client struct {
	ctx    context.Context
	userID string // The phone number registered with signal-cli (e.g., "+15551234567")
}

// NewClient creates a new Signal client for the specified phone number
// The phone number must be already registered with signal-cli
func NewClient(ctx context.Context, userID string) (*Client, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	// Validate that the phone number starts with + (required by signal-cli)
	if !strings.HasPrefix(userID, "+") {
		return nil, fmt.Errorf("userID must be a phone number starting with + (e.g., +15551234567)")
	}

	// Verify signal-cli is installed by checking if the command exists
	_, err := exec.LookPath("signal-cli")
	if err != nil {
		return nil, &SignalError{
			Op:     "initialize",
			UserID: userID,
			Err:    fmt.Errorf("signal-cli not found in PATH. Please install signal-cli: https://github.com/AsamK/signal-cli"),
		}
	}

	return &Client{
		ctx:    ctx,
		userID: userID,
	}, nil
}

// UserID returns the phone number associated with this client
func (c *Client) UserID() string {
	return c.userID
}

// runCommand executes a signal-cli command and returns stdout, stderr, and any error
func (c *Client) runCommand(args ...string) (string, string, error) {
	cmd := exec.CommandContext(c.ctx, "signal-cli", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

// SendMessage sends a text message to a Signal user
func (c *Client) SendMessage(recipient string, message string) error {
	if recipient == "" {
		return &SignalError{
			Op:     "send",
			UserID: c.userID,
			Err:    fmt.Errorf("recipient cannot be empty"),
		}
	}

	if message == "" {
		return &SignalError{
			Op:     "send",
			UserID: c.userID,
			Err:    fmt.Errorf("message cannot be empty"),
		}
	}

	// Validate recipient format
	if !strings.HasPrefix(recipient, "+") {
		return &SignalError{
			Op:     "send",
			UserID: c.userID,
			Err:    fmt.Errorf("recipient must be a phone number starting with + (e.g., +15551234567)"),
		}
	}

	// Build command: signal-cli -u USER_ID send RECIPIENT -m MESSAGE
	args := []string{"-u", c.userID, "send", recipient, "-m", message}

	_, stderr, err := c.runCommand(args...)
	if err != nil {
		return &SignalError{
			Op:     "send",
			UserID: c.userID,
			Err:    fmt.Errorf("failed to send message: %w (stderr: %s)", err, stderr),
		}
	}

	return nil
}

// SendGroupMessage sends a text message to a Signal group
func (c *Client) SendGroupMessage(groupName string, message string) error {
	if groupName == "" {
		return &SignalError{
			Op:     "sendGroup",
			UserID: c.userID,
			Err:    fmt.Errorf("groupName cannot be empty"),
		}
	}

	if message == "" {
		return &SignalError{
			Op:     "sendGroup",
			UserID: c.userID,
			Err:    fmt.Errorf("message cannot be empty"),
		}
	}

	// Verify the group exists before attempting to send
	groupID, err := c.getGroupID(groupName)
	if err != nil {
		return &SignalError{
			Op:     "sendGroup",
			UserID: c.userID,
			Err:    fmt.Errorf("group not found: %w", err),
		}
	}

	// Build command: signal-cli -u USER_ID send -g GROUP_ID -m MESSAGE
	args := []string{"-u", c.userID, "send", "-g", groupID, "-m", message}

	_, stderr, err := c.runCommand(args...)
	if err != nil {
		return &SignalError{
			Op:     "sendGroup",
			UserID: c.userID,
			Err:    fmt.Errorf("failed to send group message: %w (stderr: %s)", err, stderr),
		}
	}

	return nil
}

// getGroupID looks up a group ID by its name
func (c *Client) getGroupID(groupName string) (string, error) {
	// Build command: signal-cli -u USER_ID listGroups
	args := []string{"-u", c.userID, "listGroups"}

	stdout, stderr, err := c.runCommand(args...)
	if err != nil {
		return "", fmt.Errorf("failed to list groups: %w (stderr: %s)", err, stderr)
	}

	// Parse the output to find the group
	// signal-cli output format typically includes "Name: <group_name>"
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Name: ") && strings.Contains(line, groupName) {
			// Found the group - return the name itself as the ID
			// signal-cli can use the group name directly in send commands
			return groupName, nil
		}
	}

	return "", fmt.Errorf("group %q not found", groupName)
}

// ListGroups returns a list of all groups the user is a member of
func (c *Client) ListGroups() ([]Group, error) {
	// Build command: signal-cli -u USER_ID listGroups
	args := []string{"-u", c.userID, "listGroups"}

	stdout, stderr, err := c.runCommand(args...)
	if err != nil {
		return nil, &SignalError{
			Op:     "listGroups",
			UserID: c.userID,
			Err:    fmt.Errorf("failed to list groups: %w (stderr: %s)", err, stderr),
		}
	}

	groups := []Group{}
	var currentGroup *Group

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Parse group information from signal-cli output
		if strings.HasPrefix(line, "Id: ") {
			// Start of a new group
			if currentGroup != nil {
				groups = append(groups, *currentGroup)
			}
			currentGroup = &Group{
				ID:      strings.TrimPrefix(line, "Id: "),
				Members: []string{},
			}
		} else if strings.HasPrefix(line, "Name: ") && currentGroup != nil {
			currentGroup.Name = strings.TrimPrefix(line, "Name: ")
		}
	}

	// Add the last group if exists
	if currentGroup != nil {
		groups = append(groups, *currentGroup)
	}

	return groups, nil
}

// ReceiveMessage waits for and receives a Signal message with the specified timeout
// Returns a MessageResponse with the message details, or an empty response if no message
// was received within the timeout
func (c *Client) ReceiveMessage(timeoutSeconds int) (*MessageResponse, error) {
	if timeoutSeconds <= 0 {
		return nil, &SignalError{
			Op:     "receive",
			UserID: c.userID,
			Err:    fmt.Errorf("timeout must be positive"),
		}
	}

	// Build command: signal-cli -u USER_ID receive --timeout TIMEOUT
	args := []string{"-u", c.userID, "receive", "--timeout", strconv.Itoa(timeoutSeconds)}

	stdout, stderr, err := c.runCommand(args...)

	// Check for timeout (not an error, just no messages)
	if err != nil && strings.Contains(strings.ToLower(stderr), "timeout") {
		return &MessageResponse{}, nil
	}

	if err != nil {
		return &MessageResponse{
				Error: fmt.Sprintf("failed to receive message: %v (stderr: %s)", err, stderr),
			}, &SignalError{
				Op:     "receive",
				UserID: c.userID,
				Err:    fmt.Errorf("failed to receive message: %w (stderr: %s)", err, stderr),
			}
	}

	// If stdout is empty, no messages were received
	if strings.TrimSpace(stdout) == "" {
		return &MessageResponse{}, nil
	}

	// Parse the output
	response, err := c.parseReceiveOutput(stdout)
	if err != nil {
		return &MessageResponse{
				Error: fmt.Sprintf("failed to parse message: %v", err),
			}, &SignalError{
				Op:     "receive",
				UserID: c.userID,
				Err:    fmt.Errorf("failed to parse message: %w", err),
			}
	}

	return response, nil
}

// parseReceiveOutput parses the output from signal-cli receive command
func (c *Client) parseReceiveOutput(output string) (*MessageResponse, error) {
	response := &MessageResponse{}
	inGroup := false
	var currentSender string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Parse envelope information
		// Format: Envelope from: "Name" +11234567890 (device: 1) to +15551234567
		if strings.HasPrefix(line, "Envelope from:") {
			// Extract phone number
			parts := strings.Split(line, "+")
			if len(parts) > 1 {
				// Get the sender's phone number (first + after "Envelope from:")
				phonePart := strings.Fields(parts[1])[0]
				currentSender = "+" + phonePart
				response.SenderID = currentSender
			}
		}

		// Parse message body
		if strings.HasPrefix(line, "Body:") {
			message := strings.TrimPrefix(line, "Body:")
			message = strings.TrimSpace(message)
			response.Message = message

			// If we have both sender and message, we can return
			if response.SenderID != "" && response.Message != "" {
				return response, nil
			}
		}

		// Check if message is from a group
		if strings.HasPrefix(line, "Group info:") {
			inGroup = true
		}

		// Parse group name
		if strings.HasPrefix(line, "Name:") && inGroup {
			groupName := strings.TrimPrefix(line, "Name:")
			groupName = strings.TrimSpace(groupName)
			response.GroupName = groupName
		}
	}

	// If we didn't find a message with body, return an error
	if response.Message == "" {
		return nil, fmt.Errorf("no message body found in output")
	}

	return response, nil
}

// ReceiveMessages continuously receives messages until the context is cancelled
// It calls the provided callback function for each received message
func (c *Client) ReceiveMessages(callback func(*MessageResponse) error, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			// Try to receive a message with a reasonable timeout
			response, err := c.ReceiveMessage(int(pollInterval.Seconds()))
			if err != nil {
				// Log error but continue polling
				continue
			}

			// If we got a message, call the callback
			if response.Message != "" {
				if err := callback(response); err != nil {
					return fmt.Errorf("callback error: %w", err)
				}
			}
		}
	}
}
