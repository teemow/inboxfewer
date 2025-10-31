// Package tasks provides a client for managing Google Tasks.
//
// This package wraps the Google Tasks API (tasks/v1) and provides functionality for:
//   - Managing task lists (list, get, create, update, delete)
//   - Managing tasks (list, get, create, update, delete, complete, move)
//   - Filtering tasks by due date and completion status
//   - Creating subtasks and organizing task hierarchies
//
// The client supports both CLI and MCP server modes, with multi-account authentication
// through the unified Google OAuth2 system.
//
// # Authentication
//
// The client uses the same OAuth2 token system as other Google services (Gmail, Calendar, etc.).
// Tokens are stored per-account in the user's cache directory and provide access to:
//   - Google Tasks (read/write)
//
// For CLI usage, the client will prompt for authorization if no token exists.
// For MCP server usage, it will return an error with instructions to use the
// google_get_auth_url and google_save_auth_code tools.
//
// # Example Usage
//
//	// Create a client for the default account
//	client, err := tasks.NewClient(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List all task lists
//	lists, err := client.ListTaskLists()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create a new task
//	task, err := client.CreateTask(lists[0].ID, tasks.TaskInput{
//	    Title: "Complete project",
//	    Notes: "Finish implementation and testing",
//	    Due:   time.Now().AddDate(0, 0, 7), // Due in 7 days
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Complete a task
//	completed, err := client.CompleteTask(lists[0].ID, task.ID)
//	if err != nil {
//	    log.Fatal(err)
//	}
package tasks
