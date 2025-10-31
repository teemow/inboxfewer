// Package tasks_tools provides MCP tools for managing Google Tasks.
//
// This package implements MCP (Model Context Protocol) tools that wrap the
// Google Tasks client functionality, providing task and task list management
// capabilities for AI assistants.
//
// # Available Tools
//
// Task List Management:
//   - tasks_list_task_lists: List all task lists
//   - tasks_get_task_list: Get details of a specific task list
//   - tasks_create_task_list: Create a new task list
//   - tasks_update_task_list: Update a task list's title
//   - tasks_delete_task_list: Delete a task list
//
// Task Management:
//   - tasks_list_tasks: List tasks in a task list (with filters)
//   - tasks_get_task: Get details of a specific task
//   - tasks_create_task: Create a new task
//   - tasks_update_task: Update a task
//   - tasks_delete_task: Delete a task
//   - tasks_complete_task: Mark a task as completed
//   - tasks_move_task: Move a task to another position or parent
//   - tasks_clear_completed: Clear all completed tasks from a list
//
// # Multi-Account Support
//
// All tools support an optional 'account' parameter to specify which Google account
// to use. If not provided, the 'default' account is used.
//
// # Authentication
//
// Tools use the unified Google OAuth system. If no valid token exists for an account,
// tools will return a helpful error message with instructions to use the
// google_get_auth_url and google_save_auth_code tools.
package tasks_tools
