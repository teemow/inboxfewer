package drive_tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/drive"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/batch"
	"github.com/teemow/inboxfewer/internal/tools/common"
)

// registerShareTools registers file sharing and permission management tools
func registerShareTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Register write tools only if not in read-only mode
	if !readOnly {
		// Share files tool
		shareFilesTool := mcp.NewTool("drive_share_files",
			mcp.WithDescription("Share one or more files in Google Drive by granting permissions"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("fileIds",
				mcp.Required(),
				mcp.Description("File ID (string) or array of file IDs to share"),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("The type of grantee: 'user', 'group', 'domain', or 'anyone'"),
			),
			mcp.WithString("role",
				mcp.Required(),
				mcp.Description("The role to grant: 'owner', 'organizer', 'fileOrganizer', 'writer', 'commenter', or 'reader'"),
			),
			mcp.WithString("emailAddress",
				mcp.Description("Email address (required if type is 'user' or 'group')"),
			),
			mcp.WithString("domain",
				mcp.Description("Domain name (required if type is 'domain')"),
			),
			mcp.WithBoolean("sendNotificationEmail",
				mcp.Description("Send a notification email to the grantee (default: false)"),
			),
			mcp.WithString("emailMessage",
				mcp.Description("Custom message to include in the notification email"),
			),
		)

		s.AddTool(shareFilesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := common.GetAccountFromArgs(ctx, args)

			fileIDs, err := batch.ParseStringOrArray(args["fileIds"], "fileIds")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			permType, ok := args["type"].(string)
			if !ok || permType == "" {
				return mcp.NewToolResultError("type is required"), nil
			}

			role, ok := args["role"].(string)
			if !ok || role == "" {
				return mcp.NewToolResultError("role is required"), nil
			}

			client, err := getDriveClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			options := &drive.ShareOptions{
				Type: permType,
				Role: role,
			}

			if emailAddress, ok := args["emailAddress"].(string); ok && emailAddress != "" {
				options.EmailAddress = emailAddress
			}

			if domain, ok := args["domain"].(string); ok && domain != "" {
				options.Domain = domain
			}

			if sendNotif, ok := args["sendNotificationEmail"].(bool); ok {
				options.SendNotificationEmail = sendNotif
			}

			if emailMsg, ok := args["emailMessage"].(string); ok && emailMsg != "" {
				options.EmailMessage = emailMsg
			}

			results := batch.ProcessBatch(fileIDs, func(fileID string) (string, error) {
				permission, err := client.ShareFile(ctx, fileID, options)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("File %s shared with %s (%s)", fileID, options.EmailAddress, permission.Role), nil
			})

			return mcp.NewToolResultText(batch.FormatResults(results)), nil
		})
	}

	// List permissions tool (read-only, always available)
	listPermissionsTool := mcp.NewTool("drive_list_permissions",
		mcp.WithDescription("List all permissions for a file in Google Drive"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("fileId",
			mcp.Required(),
			mcp.Description("The ID of the file"),
		),
	)

	s.AddTool(listPermissionsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := common.GetAccountFromArgs(ctx, args)

		fileID, ok := args["fileId"].(string)
		if !ok || fileID == "" {
			return mcp.NewToolResultError("fileId is required"), nil
		}

		client, err := getDriveClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		permissions, err := client.ListPermissions(ctx, fileID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list permissions: %v", err)), nil
		}

		result, _ := json.MarshalIndent(permissions, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	})

	// Remove permission tool (write operation, only available with !readOnly)
	if !readOnly {
		removePermissionTool := mcp.NewTool("drive_remove_permission",
			mcp.WithDescription("Remove a permission from a file in Google Drive"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("fileId",
				mcp.Required(),
				mcp.Description("The ID of the file"),
			),
			mcp.WithString("permissionId",
				mcp.Required(),
				mcp.Description("The ID of the permission to remove (get this from drive_list_permissions)"),
			),
		)

		s.AddTool(removePermissionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := common.GetAccountFromArgs(ctx, args)

			fileID, ok := args["fileId"].(string)
			if !ok || fileID == "" {
				return mcp.NewToolResultError("fileId is required"), nil
			}

			permissionID, ok := args["permissionId"].(string)
			if !ok || permissionID == "" {
				return mcp.NewToolResultError("permissionId is required"), nil
			}

			client, err := getDriveClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			err = client.RemovePermission(ctx, fileID, permissionID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to remove permission: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Permission %s removed successfully from file %s", permissionID, fileID)), nil
		})
	}

	return nil
}
