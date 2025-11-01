package drive_tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/drive"
	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/batch"
)

// registerFileTools registers file management tools
func registerFileTools(s *mcpserver.MCPServer, sc *server.ServerContext, readOnly bool) error {
	// Register write tools only if not in read-only mode
	if !readOnly {
		// Upload file tool
		uploadFileTool := mcp.NewTool("drive_upload_file",
			mcp.WithDescription("Upload a file to Google Drive"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the file"),
			),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("The file content (base64-encoded for binary files, or plain text)"),
			),
			mcp.WithString("mimeType",
				mcp.Description("The MIME type of the file (e.g., 'application/pdf', 'text/plain', 'image/png')"),
			),
			mcp.WithString("parentFolders",
				mcp.Description("Comma-separated list of parent folder IDs where the file should be placed"),
			),
			mcp.WithString("description",
				mcp.Description("A short description of the file"),
			),
			mcp.WithBoolean("isBase64",
				mcp.Description("Whether the content is base64-encoded (default: true for binary files, false for text)"),
			),
		)

		s.AddTool(uploadFileTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			name, ok := args["name"].(string)
			if !ok || name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			contentStr, ok := args["content"].(string)
			if !ok || contentStr == "" {
				return mcp.NewToolResultError("content is required"), nil
			}

			client, err := getDriveClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			options := &drive.UploadOptions{}

			if mimeType, ok := args["mimeType"].(string); ok && mimeType != "" {
				options.MimeType = mimeType
			}

			if description, ok := args["description"].(string); ok && description != "" {
				options.Description = description
			}

			if parentFoldersStr, ok := args["parentFolders"].(string); ok && parentFoldersStr != "" {
				options.ParentFolders = parseCommaList(parentFoldersStr)
			}

			// Decode content if base64
			isBase64 := true
			if isB64, ok := args["isBase64"].(bool); ok {
				isBase64 = isB64
			}

			var content io.Reader
			if isBase64 {
				decoded, err := base64.StdEncoding.DecodeString(contentStr)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to decode base64 content: %v", err)), nil
				}
				content = strings.NewReader(string(decoded))
			} else {
				content = strings.NewReader(contentStr)
			}

			fileInfo, err := client.UploadFile(ctx, name, content, options)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to upload file: %v", err)), nil
			}

			result, _ := json.MarshalIndent(fileInfo, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("File uploaded successfully:\n%s", string(result))), nil
		})
	}

	// List files tool (read-only, always available)
	listFilesTool := mcp.NewTool("drive_list_files",
		mcp.WithDescription("List files in Google Drive with optional filtering"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("query",
			mcp.Description("Query for filtering files using Google Drive's query language (e.g., \"name contains 'report'\", \"mimeType='application/pdf'\")"),
		),
		mcp.WithNumber("maxResults",
			mcp.Description("Maximum number of files to return (default: 100, max: 1000)"),
		),
		mcp.WithString("orderBy",
			mcp.Description("Sort order (e.g., 'folder,modifiedTime desc,name')"),
		),
		mcp.WithBoolean("includeTrashed",
			mcp.Description("Include trashed files in results (default: false)"),
		),
		mcp.WithString("pageToken",
			mcp.Description("Page token for retrieving the next page of results"),
		),
	)

	s.AddTool(listFilesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		client, err := getDriveClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		options := &drive.ListOptions{
			MaxResults: 100, // default
		}

		if query, ok := args["query"].(string); ok && query != "" {
			options.Query = query
		}

		if maxResults, ok := args["maxResults"].(float64); ok && maxResults > 0 {
			options.MaxResults = int(maxResults)
		}

		if orderBy, ok := args["orderBy"].(string); ok && orderBy != "" {
			options.OrderBy = orderBy
		}

		if includeTrashed, ok := args["includeTrashed"].(bool); ok {
			options.IncludeTrashed = includeTrashed
		}

		if pageToken, ok := args["pageToken"].(string); ok && pageToken != "" {
			options.PageToken = pageToken
		}

		files, nextPageToken, err := client.ListFiles(ctx, options)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list files: %v", err)), nil
		}

		response := map[string]interface{}{
			"files":         files,
			"nextPageToken": nextPageToken,
		}

		result, _ := json.MarshalIndent(response, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	})

	// Get files tool
	getFilesTool := mcp.NewTool("drive_get_files",
		mcp.WithDescription("Get metadata for one or more files in Google Drive"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("fileIds",
			mcp.Required(),
			mcp.Description("File ID (string) or array of file IDs to retrieve"),
		),
	)

	s.AddTool(getFilesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		fileIDs, err := batch.ParseStringOrArray(args["fileIds"], "fileIds")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		client, err := getDriveClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		results := batch.ProcessBatch(fileIDs, func(fileID string) (string, error) {
			fileInfo, err := client.GetFile(ctx, fileID)
			if err != nil {
				return "", err
			}
			jsonBytes, _ := json.Marshal(fileInfo)
			return string(jsonBytes), nil
		})

		return mcp.NewToolResultText(batch.FormatResults(results)), nil
	})

	// Download files tool
	downloadFilesTool := mcp.NewTool("drive_download_files",
		mcp.WithDescription("Download the content of one or more files from Google Drive"),
		mcp.WithString("account",
			mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
		),
		mcp.WithString("fileIds",
			mcp.Required(),
			mcp.Description("File ID (string) or array of file IDs to download"),
		),
		mcp.WithBoolean("asBase64",
			mcp.Description("Return content as base64-encoded string (default: false for text)"),
		),
	)

	s.AddTool(downloadFilesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		account := getAccountFromArgs(args)

		fileIDs, err := batch.ParseStringOrArray(args["fileIds"], "fileIds")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		asBase64 := false
		if asB64, ok := args["asBase64"].(bool); ok {
			asBase64 = asB64
		}

		client, err := getDriveClient(ctx, account, sc)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		results := batch.ProcessBatch(fileIDs, func(fileID string) (string, error) {
			reader, err := client.DownloadFile(ctx, fileID)
			if err != nil {
				return "", err
			}
			defer reader.Close()

			content, err := io.ReadAll(reader)
			if err != nil {
				return "", fmt.Errorf("failed to read content: %w", err)
			}

			if asBase64 {
				encoded := base64.StdEncoding.EncodeToString(content)
				return fmt.Sprintf("File content (base64, %d bytes):\n%s", len(content), encoded), nil
			}

			return fmt.Sprintf("File content (text, %d bytes):\n%s", len(content), string(content)), nil
		})

		return mcp.NewToolResultText(batch.FormatResults(results)), nil
	})

	// Delete files tool (write operation, only available with !readOnly)
	if !readOnly {
		deleteFilesTool := mcp.NewTool("drive_delete_files",
			mcp.WithDescription("Delete one or more files from Google Drive"),
			mcp.WithString("account",
				mcp.Description("Account name (default: 'default'). Used to manage multiple Google accounts."),
			),
			mcp.WithString("fileIds",
				mcp.Required(),
				mcp.Description("File ID (string) or array of file IDs to delete"),
			),
		)

		s.AddTool(deleteFilesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			account := getAccountFromArgs(args)

			fileIDs, err := batch.ParseStringOrArray(args["fileIds"], "fileIds")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getDriveClient(ctx, account, sc)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			results := batch.ProcessBatch(fileIDs, func(fileID string) (string, error) {
				if err := client.DeleteFile(ctx, fileID); err != nil {
					return "", err
				}
				return fmt.Sprintf("File %s deleted successfully", fileID), nil
			})

			return mcp.NewToolResultText(batch.FormatResults(results)), nil
		})
	}

	return nil
}

// parseCommaList parses a comma-separated list of strings
func parseCommaList(s string) []string {
	if s == "" {
		return nil
	}

	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
