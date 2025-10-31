package tasks

import (
	"time"

	tasks "google.golang.org/api/tasks/v1"
)

// TaskList represents a Google Tasks task list
type TaskList struct {
	ID      string
	Title   string
	Updated time.Time
}

// Task represents a Google Tasks task
type Task struct {
	ID        string
	Title     string
	Notes     string
	Status    string // "needsAction" or "completed"
	Due       time.Time
	Completed time.Time
	Parent    string // Parent task ID for subtasks
	Position  string // Position in the list
	Links     []Link // Related links
}

// Link represents a related link in a task
type Link struct {
	Type        string // "email" or other types
	Description string
	Link        string
}

// TaskInput represents the input for creating or updating a task
type TaskInput struct {
	Title    string
	Notes    string
	Status   string // "needsAction" or "completed"
	Due      time.Time
	Parent   string // Parent task ID for subtasks
	Previous string // Previous sibling task ID for positioning
}

// toTaskList converts a Google Tasks TaskList to our TaskList type
func toTaskList(tl *tasks.TaskList) TaskList {
	if tl == nil {
		return TaskList{}
	}

	result := TaskList{
		ID:    tl.Id,
		Title: tl.Title,
	}

	if tl.Updated != "" {
		if t, err := time.Parse(time.RFC3339, tl.Updated); err == nil {
			result.Updated = t
		}
	}

	return result
}

// toTask converts a Google Tasks Task to our Task type
func toTask(t *tasks.Task) Task {
	if t == nil {
		return Task{}
	}

	result := Task{
		ID:       t.Id,
		Title:    t.Title,
		Notes:    t.Notes,
		Status:   t.Status,
		Parent:   t.Parent,
		Position: t.Position,
	}

	// Parse due date
	if t.Due != "" {
		if due, err := time.Parse(time.RFC3339, t.Due); err == nil {
			result.Due = due
		}
	}

	// Parse completed date
	if t.Completed != nil && *t.Completed != "" {
		if completed, err := time.Parse(time.RFC3339, *t.Completed); err == nil {
			result.Completed = completed
		}
	}

	// Convert links
	if t.Links != nil {
		result.Links = make([]Link, len(t.Links))
		for i, link := range t.Links {
			result.Links[i] = Link{
				Type:        link.Type,
				Description: link.Description,
				Link:        link.Link,
			}
		}
	}

	return result
}
