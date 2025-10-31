package drive_tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/drive"
	"github.com/teemow/inboxfewer/internal/server"
)

// registerFolderTools registers folder management tools
func registerFolderTools(s *mcpserver.MCPServer, sc *server.ServerContext) error {
	// Create folder tool
	createFolderTool := mcp.NewTool("drive_create_folder",
		mcp.WithDescription("Create a new folder in Google Drive"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the folder"),
		),
		mcp.WithString("parentFolders",
			mcp.Description("Comma-separated list of parent folder IDs where the folder should be created"),
		),
	)

	s.AddTool(createFolderTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		name, ok := args["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		client, err := getDriveClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var parentFolders []string
		if parentFoldersStr, ok := args["parentFolders"].(string); ok && parentFoldersStr != "" {
			parentFolders = parseCommaList(parentFoldersStr)
		}

		folderInfo, err := client.CreateFolder(ctx, name, parentFolders)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create folder: %v", err)), nil
		}

		result, _ := json.MarshalIndent(folderInfo, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Folder created successfully:\n%s", string(result))), nil
	})

	// Move/rename file tool
	moveFileTool := mcp.NewTool("drive_move_file",
		mcp.WithDescription("Move or rename a file in Google Drive"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("fileId",
			mcp.Required(),
			mcp.Description("The ID of the file to move or rename"),
		),
		mcp.WithString("newName",
			mcp.Description("The new name for the file (leave empty to keep current name)"),
		),
		mcp.WithString("addParents",
			mcp.Description("Comma-separated list of folder IDs to add as parents"),
		),
		mcp.WithString("removeParents",
			mcp.Description("Comma-separated list of folder IDs to remove as parents"),
		),
	)

	s.AddTool(moveFileTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		fileID, ok := args["fileId"].(string)
		if !ok || fileID == "" {
			return mcp.NewToolResultError("fileId is required"), nil
		}

		client, err := getDriveClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		options := &drive.MoveOptions{}

		if newName, ok := args["newName"].(string); ok && newName != "" {
			options.NewName = newName
		}

		if addParents, ok := args["addParents"].(string); ok && addParents != "" {
			options.AddParents = parseCommaList(addParents)
		}

		if removeParents, ok := args["removeParents"].(string); ok && removeParents != "" {
			options.RemoveParents = parseCommaList(removeParents)
		}

		// Check if at least one operation is specified
		if options.NewName == "" && len(options.AddParents) == 0 && len(options.RemoveParents) == 0 {
			return mcp.NewToolResultError("At least one of newName, addParents, or removeParents must be specified"), nil
		}

		fileInfo, err := client.MoveFile(ctx, fileID, options)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to move file: %v", err)), nil
		}

		result, _ := json.MarshalIndent(fileInfo, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("File moved/renamed successfully:\n%s", string(result))), nil
	})

	return nil
}
