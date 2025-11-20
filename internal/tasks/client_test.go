package tasks

import (
	"testing"
	"time"

	tasks "google.golang.org/api/tasks/v1"
)

func TestToTaskList(t *testing.T) {
	// Test with nil task list
	result := toTaskList(nil)
	if result.ID != "" {
		t.Errorf("Expected empty ID for nil task list, got %s", result.ID)
	}

	// Test with valid task list
	updated := "2025-10-31T14:00:00Z"
	tl := &tasks.TaskList{
		Id:      "test-list-id",
		Title:   "My Tasks",
		Updated: updated,
	}
	result = toTaskList(tl)

	if result.ID != "test-list-id" {
		t.Errorf("Expected ID 'test-list-id', got %s", result.ID)
	}
	if result.Title != "My Tasks" {
		t.Errorf("Expected title 'My Tasks', got %s", result.Title)
	}
	if result.Updated.IsZero() {
		t.Error("Expected non-zero updated time")
	}
}

func TestToTask(t *testing.T) {
	// Test with nil task
	result := toTask(nil)
	if result.ID != "" {
		t.Errorf("Expected empty ID for nil task, got %s", result.ID)
	}

	// Test with valid task
	due := "2025-11-07T09:00:00Z"
	completed := "2025-10-31T10:00:00Z"
	task := &tasks.Task{
		Id:        "test-task-id",
		Title:     "Complete project",
		Notes:     "Implementation notes",
		Status:    "needsAction",
		Due:       due,
		Completed: &completed,
		Parent:    "parent-task-id",
		Position:  "00000000000000000001",
		Links: []*tasks.TaskLinks{
			{
				Type:        "email",
				Description: "Related email",
				Link:        "https://mail.google.com/...",
			},
		},
	}
	result = toTask(task)

	if result.ID != "test-task-id" {
		t.Errorf("Expected ID 'test-task-id', got %s", result.ID)
	}
	if result.Title != "Complete project" {
		t.Errorf("Expected title 'Complete project', got %s", result.Title)
	}
	if result.Notes != "Implementation notes" {
		t.Errorf("Expected notes 'Implementation notes', got %s", result.Notes)
	}
	if result.Status != "needsAction" {
		t.Errorf("Expected status 'needsAction', got %s", result.Status)
	}
	if result.Due.IsZero() {
		t.Error("Expected non-zero due date")
	}
	if result.Completed.IsZero() {
		t.Error("Expected non-zero completed date")
	}
	if result.Parent != "parent-task-id" {
		t.Errorf("Expected parent 'parent-task-id', got %s", result.Parent)
	}
	if len(result.Links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(result.Links))
	} else {
		if result.Links[0].Type != "email" {
			t.Errorf("Expected link type 'email', got %s", result.Links[0].Type)
		}
	}
}

func TestHasToken(t *testing.T) {
	// Test that HasToken returns a boolean without error
	result := HasToken()
	// We don't care about the actual value, just that it doesn't panic
	_ = result
}

func TestHasTokenForAccount(t *testing.T) {
	// Test that HasTokenForAccount returns a boolean for valid account name
	result := HasTokenForAccount("test-account")
	_ = result

	// Test with empty account name
	result = HasTokenForAccount("")
	if result {
		t.Error("Expected false for empty account name")
	}
}

func TestTaskInput_Validation(t *testing.T) {
	// Test TaskInput structure with various valid inputs
	tests := []struct {
		name  string
		input TaskInput
	}{
		{
			name: "valid basic task",
			input: TaskInput{
				Title:  "Buy groceries",
				Status: "needsAction",
			},
		},
		{
			name: "task with notes",
			input: TaskInput{
				Title:  "Write report",
				Notes:  "Include Q4 metrics and analysis",
				Status: "needsAction",
			},
		},
		{
			name: "task with due date",
			input: TaskInput{
				Title:  "Submit proposal",
				Due:    time.Now().AddDate(0, 0, 7),
				Status: "needsAction",
			},
		},
		{
			name: "subtask",
			input: TaskInput{
				Title:  "Review draft",
				Parent: "parent-task-id",
				Status: "needsAction",
			},
		},
		{
			name: "task with positioning",
			input: TaskInput{
				Title:    "First task",
				Previous: "other-task-id",
				Status:   "needsAction",
			},
		},
		{
			name: "completed task",
			input: TaskInput{
				Title:  "Finished task",
				Status: "completed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the structure is valid
			if tt.input.Title == "" {
				t.Error("Title should not be empty")
			}
			if tt.input.Status != "" && tt.input.Status != "needsAction" && tt.input.Status != "completed" {
				t.Errorf("Invalid status: %s", tt.input.Status)
			}
		})
	}
}

func TestClient_Account(t *testing.T) {
	// Test that Account() returns the correct account name
	c := &Client{account: "test-account"}
	if c.Account() != "test-account" {
		t.Errorf("Expected account 'test-account', got %s", c.Account())
	}
}

func TestTaskList_Fields(t *testing.T) {
	// Test TaskList structure
	tl := TaskList{
		ID:      "list-1",
		Title:   "Work Tasks",
		Updated: time.Now(),
	}

	if tl.ID != "list-1" {
		t.Errorf("Expected ID 'list-1', got %s", tl.ID)
	}
	if tl.Title != "Work Tasks" {
		t.Errorf("Expected title 'Work Tasks', got %s", tl.Title)
	}
	if tl.Updated.IsZero() {
		t.Error("Expected non-zero updated time")
	}
}

func TestTask_Fields(t *testing.T) {
	// Test Task structure
	due := time.Now().AddDate(0, 0, 1)
	completed := time.Now()

	task := Task{
		ID:        "task-1",
		Title:     "Review PR",
		Notes:     "Check tests and documentation",
		Status:    "completed",
		Due:       due,
		Completed: completed,
		Parent:    "parent-1",
		Position:  "00000000000000000001",
		Links: []Link{
			{
				Type:        "email",
				Description: "PR notification",
				Link:        "https://github.com/...",
			},
		},
	}

	if task.ID != "task-1" {
		t.Errorf("Expected ID 'task-1', got %s", task.ID)
	}
	if task.Title != "Review PR" {
		t.Errorf("Expected title 'Review PR', got %s", task.Title)
	}
	if task.Notes != "Check tests and documentation" {
		t.Errorf("Expected specific notes, got %s", task.Notes)
	}
	if task.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", task.Status)
	}
	if task.Due.IsZero() {
		t.Error("Expected non-zero due date")
	}
	if task.Completed.IsZero() {
		t.Error("Expected non-zero completed date")
	}
	if task.Parent != "parent-1" {
		t.Errorf("Expected parent 'parent-1', got %s", task.Parent)
	}
	if len(task.Links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(task.Links))
	}
}

func TestLink_Fields(t *testing.T) {
	// Test Link structure
	link := Link{
		Type:        "email",
		Description: "Related message",
		Link:        "https://example.com",
	}

	if link.Type != "email" {
		t.Errorf("Expected type 'email', got %s", link.Type)
	}
	if link.Description != "Related message" {
		t.Errorf("Expected specific description, got %s", link.Description)
	}
	if link.Link != "https://example.com" {
		t.Errorf("Expected specific link, got %s", link.Link)
	}
}

func TestToTask_EmptyDates(t *testing.T) {
	// Test task conversion with empty dates
	task := &tasks.Task{
		Id:     "task-1",
		Title:  "Task without dates",
		Status: "needsAction",
	}
	result := toTask(task)

	if !result.Due.IsZero() {
		t.Error("Expected zero due date")
	}
	if !result.Completed.IsZero() {
		t.Error("Expected zero completed date")
	}
}

func TestToTask_InvalidDates(t *testing.T) {
	// Test task conversion with invalid date formats
	invalidCompleted := "also-not-a-date"
	task := &tasks.Task{
		Id:        "task-1",
		Title:     "Task with invalid dates",
		Due:       "not-a-date",
		Completed: &invalidCompleted,
	}
	result := toTask(task)

	// Should gracefully handle invalid dates by keeping them zero
	if !result.Due.IsZero() {
		t.Error("Expected zero due date for invalid format")
	}
	if !result.Completed.IsZero() {
		t.Error("Expected zero completed date for invalid format")
	}
}

func TestToTaskList_InvalidDate(t *testing.T) {
	// Test task list conversion with invalid date format
	tl := &tasks.TaskList{
		Id:      "list-1",
		Title:   "Test List",
		Updated: "not-a-date",
	}
	result := toTaskList(tl)

	// Should gracefully handle invalid date by keeping it zero
	if !result.Updated.IsZero() {
		t.Error("Expected zero updated time for invalid format")
	}
}

func TestToTask_NilLinks(t *testing.T) {
	// Test task conversion with nil links
	task := &tasks.Task{
		Id:    "task-1",
		Title: "Task without links",
		Links: nil,
	}
	result := toTask(task)

	if result.Links != nil {
		t.Error("Expected nil links")
	}
}

func TestToTask_EmptyLinks(t *testing.T) {
	// Test task conversion with empty links slice
	task := &tasks.Task{
		Id:    "task-1",
		Title: "Task with empty links",
		Links: []*tasks.TaskLinks{},
	}
	result := toTask(task)

	if result.Links == nil {
		t.Error("Expected non-nil links slice")
	}
	if len(result.Links) != 0 {
		t.Errorf("Expected 0 links, got %d", len(result.Links))
	}
}
