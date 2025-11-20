package tasks

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/api/option"
	tasks "google.golang.org/api/tasks/v1"

	"github.com/teemow/inboxfewer/internal/google"
)

// Client wraps the Google Tasks service
type Client struct {
	svc     *tasks.Service
	account string // The account this client is associated with
}

// Account returns the account name this client is associated with
func (c *Client) Account() string {
	return c.account
}

// HasTokenForAccount checks if a valid OAuth token exists for the specified account
func HasTokenForAccount(account string) bool {
	return google.HasTokenForAccount(account)
}

// HasToken checks if a valid OAuth token exists for the default account
func HasToken() bool {
	return google.HasToken()
}

// NewClientForAccount creates a new Tasks client with OAuth2 authentication for a specific account
// The OAuth token must be provided by the MCP client through the OAuth middleware
func NewClientForAccount(ctx context.Context, account string) (*Client, error) {
	// Get HTTP client with OAuth token
	client, err := google.GetHTTPClientForAccount(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("no valid Google OAuth token found for account %s: %w. Please authenticate with Google through your MCP client", account, err)
	}

	svc, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Tasks service: %w", err)
	}

	return &Client{
		svc:     svc,
		account: account,
	}, nil
}

// NewClient creates a new Tasks client with OAuth2 authentication for the default account
// For CLI usage, it will prompt for auth code via stdin if no token exists
// For MCP usage, it will return an error if no token exists
func NewClient(ctx context.Context) (*Client, error) {
	return NewClientForAccount(ctx, "default")
}

// isTerminal checks if stdin is connected to a terminal (CLI mode)
func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ListTaskLists lists all task lists for the authenticated user
func (c *Client) ListTaskLists() ([]TaskList, error) {
	result, err := c.svc.Tasklists.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list task lists: %w", err)
	}

	var taskLists []TaskList
	for _, tl := range result.Items {
		taskLists = append(taskLists, toTaskList(tl))
	}

	return taskLists, nil
}

// GetTaskList retrieves a specific task list by ID
func (c *Client) GetTaskList(taskListID string) (*TaskList, error) {
	tl, err := c.svc.Tasklists.Get(taskListID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get task list: %w", err)
	}

	result := toTaskList(tl)
	return &result, nil
}

// CreateTaskList creates a new task list
func (c *Client) CreateTaskList(title string) (*TaskList, error) {
	tl := &tasks.TaskList{
		Title: title,
	}

	created, err := c.svc.Tasklists.Insert(tl).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create task list: %w", err)
	}

	result := toTaskList(created)
	return &result, nil
}

// DeleteTaskList deletes a task list
func (c *Client) DeleteTaskList(taskListID string) error {
	err := c.svc.Tasklists.Delete(taskListID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete task list: %w", err)
	}
	return nil
}

// UpdateTaskList updates a task list's title
func (c *Client) UpdateTaskList(taskListID, title string) (*TaskList, error) {
	tl := &tasks.TaskList{
		Title: title,
	}

	updated, err := c.svc.Tasklists.Update(taskListID, tl).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update task list: %w", err)
	}

	result := toTaskList(updated)
	return &result, nil
}

// ListTasks lists tasks in a task list
// Options:
// - showCompleted: if true, includes completed tasks
// - dueMin: only tasks with due date after this time
// - dueMax: only tasks with due date before this time
func (c *Client) ListTasks(taskListID string, showCompleted bool, dueMin, dueMax time.Time) ([]Task, error) {
	call := c.svc.Tasks.List(taskListID)

	// Set show completed
	if showCompleted {
		call = call.ShowCompleted(true)
	}

	// Set due date filters if provided
	if !dueMin.IsZero() {
		call = call.DueMin(dueMin.Format(time.RFC3339))
	}
	if !dueMax.IsZero() {
		call = call.DueMax(dueMax.Format(time.RFC3339))
	}

	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	var taskList []Task
	for _, t := range result.Items {
		taskList = append(taskList, toTask(t))
	}

	return taskList, nil
}

// GetTask retrieves a specific task by ID
func (c *Client) GetTask(taskListID, taskID string) (*Task, error) {
	t, err := c.svc.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	result := toTask(t)
	return &result, nil
}

// CreateTask creates a new task
func (c *Client) CreateTask(taskListID string, input TaskInput) (*Task, error) {
	t := &tasks.Task{
		Title:  input.Title,
		Notes:  input.Notes,
		Status: input.Status,
	}

	// Set parent for subtasks
	if input.Parent != "" {
		t.Parent = input.Parent
	}

	// Set due date if provided
	if !input.Due.IsZero() {
		t.Due = input.Due.Format(time.RFC3339)
	}

	// Create the task
	call := c.svc.Tasks.Insert(taskListID, t)

	// Set position if previous sibling is specified
	if input.Previous != "" {
		call = call.Previous(input.Previous)
	}

	// Set parent if specified
	if input.Parent != "" {
		call = call.Parent(input.Parent)
	}

	created, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	result := toTask(created)
	return &result, nil
}

// UpdateTask updates an existing task
func (c *Client) UpdateTask(taskListID, taskID string, input TaskInput) (*Task, error) {
	// Get existing task first
	existing, err := c.svc.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing task: %w", err)
	}

	// Update fields
	if input.Title != "" {
		existing.Title = input.Title
	}
	if input.Notes != "" {
		existing.Notes = input.Notes
	}
	if input.Status != "" {
		existing.Status = input.Status
	}
	if !input.Due.IsZero() {
		existing.Due = input.Due.Format(time.RFC3339)
	}

	// Update the task
	updated, err := c.svc.Tasks.Update(taskListID, taskID, existing).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	result := toTask(updated)
	return &result, nil
}

// DeleteTask deletes a task
func (c *Client) DeleteTask(taskListID, taskID string) error {
	err := c.svc.Tasks.Delete(taskListID, taskID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// CompleteTask marks a task as completed
func (c *Client) CompleteTask(taskListID, taskID string) (*Task, error) {
	// Get existing task
	existing, err := c.svc.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Mark as completed
	existing.Status = "completed"
	completedTime := time.Now().Format(time.RFC3339)
	existing.Completed = &completedTime

	// Update the task
	updated, err := c.svc.Tasks.Update(taskListID, taskID, existing).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	result := toTask(updated)
	return &result, nil
}

// MoveTask moves a task to a different position or parent
func (c *Client) MoveTask(taskListID, taskID string, parent, previous string) (*Task, error) {
	call := c.svc.Tasks.Move(taskListID, taskID)

	// Set parent if specified (for making it a subtask or moving to root)
	if parent != "" {
		call = call.Parent(parent)
	}

	// Set previous sibling if specified (for ordering)
	if previous != "" {
		call = call.Previous(previous)
	}

	moved, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to move task: %w", err)
	}

	result := toTask(moved)
	return &result, nil
}

// ClearCompletedTasks clears all completed tasks from a task list
func (c *Client) ClearCompletedTasks(taskListID string) error {
	err := c.svc.Tasks.Clear(taskListID).Do()
	if err != nil {
		return fmt.Errorf("failed to clear completed tasks: %w", err)
	}
	return nil
}
