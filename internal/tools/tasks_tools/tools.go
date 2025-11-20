package tasks_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/google"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tasks"
	"github.com/teemow/inboxfewer/internal/tools/batch"
)

// getAccountFromArgs extracts the account name from request arguments, defaulting to "default"
func getAccountFromArgs(args map[string]interface{}) string {
	account := "default"
	if accountVal, ok := args["account"].(string); ok && accountVal != "" {
		account = accountVal
	}
	return account
}

// getTasksClient retrieves or creates a tasks client for the specified account
func getTasksClient(ctx context.Context, account string, sc *server.ServerContext) (*tasks.Client, error) {
	client := sc.TasksClientForAccount(account)
	if client == nil {
		// Check if token exists before trying to create client
		if !tasks.HasTokenForAccount(account) {
			authURL := google.GetAuthenticationErrorMessage(account)
			return nil, fmt.Errorf(`Google OAuth token not found for account "%s". To authorize access:

1. Visit this URL in your browser:
   %s

2. Sign in with your Google account
3. Grant access to Google services (Tasks, Calendar, Gmail, Docs, Drive)
4. Copy the authorization code

5. Provide the authorization code to your AI agent
   The agent will use the google_save_auth_code tool with account="%s" to complete authentication.

Note: You only need to authorize once. The tokens will be automatically refreshed.`, account, authURL, account)
		}

		var err error
		client, err = tasks.NewClientForAccount(ctx, account)
		if err != nil {
			return nil, fmt.Errorf("failed to create Tasks client for account %s: %w", account, err)
		}
		sc.SetTasksClientForAccount(account, client)
	}
	return client, nil
}

// RegisterTasksTools registers all Tasks-related tools with the MCP server
func RegisterTasksTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Register task list tools (some operations require !readOnly)
	if err := registerTaskListTools(s, sc, readOnly); err != nil {
		return fmt.Errorf("failed to register task list tools: %w", err)
	}

	// Register task tools (some operations require !readOnly)
	if err := registerTaskTools(s, sc, readOnly); err != nil {
		return fmt.Errorf("failed to register task tools: %w", err)
	}

	return nil
}

// registerTaskListTools registers task list management tools
func registerTaskListTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// List task lists tool (read-only, always available)
	listTaskListsTool := mcp.NewTool("tasks_list_task_lists",
		mcp.WithDescription("List all task lists for the authenticated user"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
	)

	s.AddTool(listTaskListsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		client, err := getTasksClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		lists, err := client.ListTaskLists()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list task lists: %v", err)), nil
		}

		result, _ := json.MarshalIndent(lists, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	})

	// Get task list tool
	getTaskListTool := mcp.NewTool("tasks_get_task_list",
		mcp.WithDescription("Get details of a specific task list"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("taskListId",
			mcp.Required(),
			mcp.Description("The ID of the task list to retrieve"),
		),
	)

	s.AddTool(getTaskListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		taskListID, ok := args["taskListId"].(string)
		if !ok || taskListID == "" {
			return mcp.NewToolResultError("taskListId is required"), nil
		}

		client, err := getTasksClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		taskList, err := client.GetTaskList(taskListID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get task list: %v", err)), nil
		}

		result, _ := json.MarshalIndent(taskList, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	})

	// Create task list tool
	createTaskListTool := mcp.NewTool("tasks_create_task_list",
		mcp.WithDescription("Create a new task list"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("The title of the new task list"),
		),
	)

	s.AddTool(createTaskListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		title, ok := args["title"].(string)
		if !ok || title == "" {
			return mcp.NewToolResultError("title is required"), nil
		}

		client, err := getTasksClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		taskList, err := client.CreateTaskList(title)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create task list: %v", err)), nil
		}

		result, _ := json.MarshalIndent(taskList, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Task list created successfully:\n%s", string(result))), nil
	})

	// Register update/delete task list tools only if not in read-only mode
	if !readOnly {
		// Update task list tool
		updateTaskListTool := mcp.NewTool("tasks_update_task_list",
			mcp.WithDescription("Update a task list's title"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("taskListId",
				mcp.Required(),
				mcp.Description("The ID of the task list to update"),
			),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("The new title for the task list"),
			),
		)

		s.AddTool(updateTaskListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			taskListID, ok := args["taskListID"].(string)
			if !ok || taskListID == "" {
				return mcp.NewToolResultError("taskListId is required"), nil
			}

			title, ok := args["title"].(string)
			if !ok || title == "" {
				return mcp.NewToolResultError("title is required"), nil
			}

			client, err := getTasksClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			taskList, err := client.UpdateTaskList(taskListID, title)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to update task list: %v", err)), nil
			}

			result, _ := json.MarshalIndent(taskList, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Task list updated successfully:\n%s", string(result))), nil
		})

		// Delete task list tool
		deleteTaskListTool := mcp.NewTool("tasks_delete_task_list",
			mcp.WithDescription("Delete a task list"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("taskListId",
				mcp.Required(),
				mcp.Description("The ID of the task list to delete"),
			),
		)

		s.AddTool(deleteTaskListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			taskListID, ok := args["taskListId"].(string)
			if !ok || taskListID == "" {
				return mcp.NewToolResultError("taskListId is required"), nil
			}

			client, err := getTasksClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			err = client.DeleteTaskList(taskListID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to delete task list: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Task list %s deleted successfully", taskListID)), nil
		})
	}

	return nil
}

// registerTaskTools registers task management tools
func registerTaskTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// List tasks tool
	listTasksTool := mcp.NewTool("tasks_list_tasks",
		mcp.WithDescription("List tasks in a task list with optional filters"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("taskListId",
			mcp.Required(),
			mcp.Description("The ID of the task list"),
		),
		mcp.WithBoolean("showCompleted",
			mcp.Description("Include completed tasks (default: false)"),
		),
		mcp.WithString("dueMin",
			mcp.Description("Only return tasks with due date after this time (RFC3339 format)"),
		),
		mcp.WithString("dueMax",
			mcp.Description("Only return tasks with due date before this time (RFC3339 format)"),
		),
	)

	s.AddTool(listTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		taskListID, ok := args["taskListId"].(string)
		if !ok || taskListID == "" {
			return mcp.NewToolResultError("taskListId is required"), nil
		}

		showCompleted := false
		if sc, ok := args["showCompleted"].(bool); ok {
			showCompleted = sc
		}

		var dueMin, dueMax time.Time
		if dueMinStr, ok := args["dueMin"].(string); ok && dueMinStr != "" {
			if t, err := time.Parse(time.RFC3339, dueMinStr); err == nil {
				dueMin = t
			}
		}
		if dueMaxStr, ok := args["dueMax"].(string); ok && dueMaxStr != "" {
			if t, err := time.Parse(time.RFC3339, dueMaxStr); err == nil {
				dueMax = t
			}
		}

		client, err := getTasksClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tasksList, err := client.ListTasks(taskListID, showCompleted, dueMin, dueMax)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list tasks: %v", err)), nil
		}

		result, _ := json.MarshalIndent(tasksList, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	})

	// Get tasks tool
	getTasksTool := mcp.NewTool("tasks_get_tasks",
		mcp.WithDescription("Get details of one or more tasks"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("taskListId",
			mcp.Required(),
			mcp.Description("The ID of the task list"),
		),
		mcp.WithString("taskIds",
			mcp.Required(),
			mcp.Description("Task ID (string) or array of task IDs to retrieve"),
		),
	)

	s.AddTool(getTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		taskListID, ok := args["taskListId"].(string)
		if !ok || taskListID == "" {
			return mcp.NewToolResultError("taskListId is required"), nil
		}

		taskIDs, err := batch.ParseStringOrArray(args["taskIds"], "taskIds")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		client, err := getTasksClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		results := batch.ProcessBatch(taskIDs, func(taskID string) (string, error) {
			task, err := client.GetTask(taskListID, taskID)
			if err != nil {
				return "", err
			}
			jsonBytes, _ := json.Marshal(task)
			return string(jsonBytes), nil
		})

		return mcp.NewToolResultText(batch.FormatResults(results)), nil
	})

	// Create tasks tool
	createTasksTool := mcp.NewTool("tasks_create_tasks",
		mcp.WithDescription("Create one or more tasks in a task list"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("taskListId",
			mcp.Required(),
			mcp.Description("The ID of the task list"),
		),
		mcp.WithString("title",
			mcp.Description("Task title (for single task creation)"),
		),
		mcp.WithString("titles",
			mcp.Description("Array of task titles (for batch task creation)"),
		),
		mcp.WithString("notes",
			mcp.Description("Notes or description for the task (single task only)"),
		),
		mcp.WithString("due",
			mcp.Description("Due date for the task (RFC3339 format, single task only)"),
		),
		mcp.WithString("parent",
			mcp.Description("Parent task ID to create a subtask (single task only)"),
		),
		mcp.WithString("previous",
			mcp.Description("Previous sibling task ID for positioning (single task only)"),
		),
	)

	s.AddTool(createTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		taskListID, ok := args["taskListId"].(string)
		if !ok || taskListID == "" {
			return mcp.NewToolResultError("taskListId is required"), nil
		}

		client, err := getTasksClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Check if batch mode (titles array) or single mode (title string)
		var titles []string
		if titlesArg, ok := args["titles"]; ok {
			parsedTitles, err := batch.ParseStringOrArray(titlesArg, "titles")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			titles = parsedTitles
		} else if title, ok := args["title"].(string); ok && title != "" {
			titles = []string{title}
		} else {
			return mcp.NewToolResultError("either 'title' or 'titles' is required"), nil
		}

		// For batch mode with simple titles, create tasks with just titles
		if len(titles) > 1 || (len(titles) == 1 && args["titles"] != nil) {
			results := batch.ProcessBatch(titles, func(title string) (string, error) {
				input := tasks.TaskInput{
					Title:  title,
					Status: "needsAction",
				}
				task, err := client.CreateTask(taskListID, input)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("Task '%s' created with ID: %s", task.Title, task.ID), nil
			})
			return mcp.NewToolResultText(batch.FormatResults(results)), nil
		}

		// Single task creation with full parameters
		input := tasks.TaskInput{
			Title:  titles[0],
			Status: "needsAction",
		}

		if notes, ok := args["notes"].(string); ok {
			input.Notes = notes
		}

		if dueStr, ok := args["due"].(string); ok && dueStr != "" {
			if due, err := time.Parse(time.RFC3339, dueStr); err == nil {
				input.Due = due
			}
		}

		if parent, ok := args["parent"].(string); ok {
			input.Parent = parent
		}

		if previous, ok := args["previous"].(string); ok {
			input.Previous = previous
		}

		task, err := client.CreateTask(taskListID, input)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create task: %v", err)), nil
		}

		result, _ := json.MarshalIndent(task, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Task created successfully:\n%s", string(result))), nil
	})

	// Register update/delete/complete/move/clear tools only if not in read-only mode
	if !readOnly {
		// Update task tool
		updateTaskTool := mcp.NewTool("tasks_update_task",
			mcp.WithDescription("Update an existing task"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("taskListId",
				mcp.Required(),
				mcp.Description("The ID of the task list"),
			),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("The ID of the task to update"),
			),
			mcp.WithString("title",
				mcp.Description("New title for the task"),
			),
			mcp.WithString("notes",
				mcp.Description("New notes or description for the task"),
			),
			mcp.WithString("status",
				mcp.Description("New status: 'needsAction' or 'completed'"),
			),
			mcp.WithString("due",
				mcp.Description("New due date for the task (RFC3339 format)"),
			),
		)

		s.AddTool(updateTaskTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			taskListID, ok := args["taskListId"].(string)
			if !ok || taskListID == "" {
				return mcp.NewToolResultError("taskListId is required"), nil
			}

			taskID, ok := args["taskId"].(string)
			if !ok || taskID == "" {
				return mcp.NewToolResultError("taskId is required"), nil
			}

			input := tasks.TaskInput{}

			if title, ok := args["title"].(string); ok {
				input.Title = title
			}

			if notes, ok := args["notes"].(string); ok {
				input.Notes = notes
			}

			if status, ok := args["status"].(string); ok {
				input.Status = status
			}

			if dueStr, ok := args["due"].(string); ok && dueStr != "" {
				if due, err := time.Parse(time.RFC3339, dueStr); err == nil {
					input.Due = due
				}
			}

			client, err := getTasksClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			task, err := client.UpdateTask(taskListID, taskID, input)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to update task: %v", err)), nil
			}

			result, _ := json.MarshalIndent(task, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Task updated successfully:\n%s", string(result))), nil
		})

		// Delete tasks tool
		deleteTasksTool := mcp.NewTool("tasks_delete_tasks",
			mcp.WithDescription("Delete one or more tasks"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("taskListId",
				mcp.Required(),
				mcp.Description("The ID of the task list"),
			),
			mcp.WithString("taskIds",
				mcp.Required(),
				mcp.Description("Task ID (string) or array of task IDs to delete"),
			),
		)

		s.AddTool(deleteTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			taskListID, ok := args["taskListId"].(string)
			if !ok || taskListID == "" {
				return mcp.NewToolResultError("taskListId is required"), nil
			}

			taskIDs, err := batch.ParseStringOrArray(args["taskIds"], "taskIds")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getTasksClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			results := batch.ProcessBatch(taskIDs, func(taskID string) (string, error) {
				if err := client.DeleteTask(taskListID, taskID); err != nil {
					return "", err
				}
				return fmt.Sprintf("Task %s deleted successfully", taskID), nil
			})

			return mcp.NewToolResultText(batch.FormatResults(results)), nil
		})

		// Complete tasks tool
		completeTasksTool := mcp.NewTool("tasks_complete_tasks",
			mcp.WithDescription("Mark one or more tasks as completed"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("taskListId",
				mcp.Required(),
				mcp.Description("The ID of the task list"),
			),
			mcp.WithString("taskIds",
				mcp.Required(),
				mcp.Description("Task ID (string) or array of task IDs to complete"),
			),
		)

		s.AddTool(completeTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			taskListID, ok := args["taskListId"].(string)
			if !ok || taskListID == "" {
				return mcp.NewToolResultError("taskListId is required"), nil
			}

			taskIDs, err := batch.ParseStringOrArray(args["taskIds"], "taskIds")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getTasksClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			results := batch.ProcessBatch(taskIDs, func(taskID string) (string, error) {
				task, err := client.CompleteTask(taskListID, taskID)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("Task %s (%s) completed successfully", taskID, task.Title), nil
			})

			return mcp.NewToolResultText(batch.FormatResults(results)), nil
		})

		// Move task tool
		moveTaskTool := mcp.NewTool("tasks_move_task",
			mcp.WithDescription("Move a task to a different position or parent"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("taskListId",
				mcp.Required(),
				mcp.Description("The ID of the task list"),
			),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("The ID of the task to move"),
			),
			mcp.WithString("parent",
				mcp.Description("New parent task ID (empty string to move to root level)"),
			),
			mcp.WithString("previous",
				mcp.Description("Previous sibling task ID for positioning"),
			),
		)

		s.AddTool(moveTaskTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			taskListID, ok := args["taskListId"].(string)
			if !ok || taskListID == "" {
				return mcp.NewToolResultError("taskListId is required"), nil
			}

			taskID, ok := args["taskId"].(string)
			if !ok || taskID == "" {
				return mcp.NewToolResultError("taskId is required"), nil
			}

			parent := ""
			if p, ok := args["parent"].(string); ok {
				parent = p
			}

			previous := ""
			if p, ok := args["previous"].(string); ok {
				previous = p
			}

			client, err := getTasksClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			task, err := client.MoveTask(taskListID, taskID, parent, previous)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to move task: %v", err)), nil
			}

			result, _ := json.MarshalIndent(task, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Task moved successfully:\n%s", string(result))), nil
		})

		// Clear completed tasks tool
		clearCompletedTool := mcp.NewTool("tasks_clear_completed",
			mcp.WithDescription("Clear all completed tasks from a task list"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("taskListId",
				mcp.Required(),
				mcp.Description("The ID of the task list to clear completed tasks from"),
			),
		)

		s.AddTool(clearCompletedTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			taskListID, ok := args["taskListId"].(string)
			if !ok || taskListID == "" {
				return mcp.NewToolResultError("taskListId is required"), nil
			}

			client, err := getTasksClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			err = client.ClearCompletedTasks(taskListID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to clear completed tasks: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Completed tasks cleared from list %s", taskListID)), nil
		})
	}

	return nil
}

// parseAttendees parses a comma-separated list of email addresses
func parseAttendees(attendeesStr string) []string {
	if attendeesStr == "" {
		return nil
	}

	var attendees []string
	for _, email := range strings.Split(attendeesStr, ",") {
		email = strings.TrimSpace(email)
		if email != "" {
			attendees = append(attendees, email)
		}
	}
	return attendees
}
