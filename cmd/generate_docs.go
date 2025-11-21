package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/teemow/inboxfewer/internal/server"
	"github.com/teemow/inboxfewer/internal/tools/calendar_tools"
	"github.com/teemow/inboxfewer/internal/tools/docs_tools"
	"github.com/teemow/inboxfewer/internal/tools/drive_tools"
	"github.com/teemow/inboxfewer/internal/tools/gmail_tools"
	"github.com/teemow/inboxfewer/internal/tools/meet_tools"
	"github.com/teemow/inboxfewer/internal/tools/tasks_tools"
)

func newGenerateDocsCmd() *cobra.Command {
	var (
		outputFile string
	)

	cmd := &cobra.Command{
		Use:   "generate-docs",
		Short: "Generate MCP tool documentation",
		Long: `Generate markdown documentation for all available MCP tools.
This command introspects the registered tools and outputs their documentation
in markdown format, ensuring the documentation is always accurate and in sync
with the actual tool implementations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateDocs(outputFile)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")

	return cmd
}

func runGenerateDocs(outputFile string) error {
	// Create a temporary server context (we don't need real credentials for doc generation)
	ctx := context.Background()
	serverContext, err := server.NewServerContext(ctx, "dummy-user", "dummy-token")
	if err != nil {
		return fmt.Errorf("failed to create server context: %w", err)
	}
	defer func() {
		_ = serverContext.Shutdown()
	}()

	// Create MCP server
	// Note: mcp.Implementation has Title field but WithTitle() ServerOption not available in v0.43.0
	mcpSrv := mcpserver.NewMCPServer("inboxfewer", version,
		mcpserver.WithToolCapabilities(true),
	)

	// Register all tools (both read-only and write modes to get all tools)
	readOnly := false // Get all tools including write operations

	// Register all tool groups
	if err := gmail_tools.RegisterGmailTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Gmail tools: %w", err)
	}

	if err := docs_tools.RegisterDocsTools(mcpSrv, serverContext); err != nil {
		return fmt.Errorf("failed to register Docs tools: %w", err)
	}

	if err := drive_tools.RegisterDriveTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Drive tools: %w", err)
	}

	if err := calendar_tools.RegisterCalendarTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Calendar tools: %w", err)
	}

	if err := meet_tools.RegisterMeetTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Meet tools: %w", err)
	}

	if err := tasks_tools.RegisterTasksTools(mcpSrv, serverContext, readOnly); err != nil {
		return fmt.Errorf("failed to register Tasks tools: %w", err)
	}

	// Get the list of tools
	serverTools := mcpSrv.ListTools()

	// Extract mcp.Tool from each ServerTool
	tools := make([]mcp.Tool, 0, len(serverTools))
	for _, serverTool := range serverTools {
		tools = append(tools, serverTool.Tool)
	}

	// Generate markdown documentation
	markdown := generateToolsMarkdown(tools)

	// Write to output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Documentation written to: %s\n", outputFile)
	} else {
		fmt.Print(markdown)
	}

	return nil
}

func generateToolsMarkdown(tools []mcp.Tool) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# MCP Tools Reference\n\n")
	sb.WriteString("This document provides a complete reference of all tools available when running inboxfewer as an MCP server.\n\n")
	sb.WriteString("**Note:** This documentation is automatically generated from the tool definitions.\n\n")

	// Group tools by category
	toolsByCategory := groupToolsByCategory(tools)

	// Table of contents
	sb.WriteString("## Table of Contents\n\n")
	categories := make([]string, 0, len(toolsByCategory))
	for category := range toolsByCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		anchor := strings.ToLower(strings.ReplaceAll(category, " ", "-"))
		sb.WriteString(fmt.Sprintf("- [%s](#%s)\n", category, anchor))
	}
	sb.WriteString("\n")

	// Multi-account support note
	sb.WriteString("## Multi-Account Support\n\n")
	sb.WriteString("All Google-related MCP tools support an optional `account` parameter to specify which Google account to use:\n\n")
	sb.WriteString("- **Default behavior:** If `account` is not specified, the `default` account is used\n")
	sb.WriteString("- **Multiple accounts:** You can manage multiple Google accounts (e.g., `work`, `personal`)\n")
	sb.WriteString("- **Per-tool specification:** Each tool call can use a different account\n\n")

	// Generate documentation for each category
	for _, category := range categories {
		categoryTools := toolsByCategory[category]
		sort.Slice(categoryTools, func(i, j int) bool {
			return categoryTools[i].Name < categoryTools[j].Name
		})

		sb.WriteString(fmt.Sprintf("## %s\n\n", category))

		for _, tool := range categoryTools {
			sb.WriteString(generateToolMarkdown(tool))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func groupToolsByCategory(tools []mcp.Tool) map[string][]mcp.Tool {
	categories := make(map[string][]mcp.Tool)

	for _, tool := range tools {
		category := getCategoryFromToolName(tool.Name)
		categories[category] = append(categories[category], tool)
	}

	return categories
}

func getCategoryFromToolName(name string) string {
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return "Other"
	}

	prefix := parts[0]
	switch prefix {
	case "gmail":
		return "Gmail Tools"
	case "docs":
		return "Google Docs Tools"
	case "drive":
		return "Google Drive Tools"
	case "calendar":
		return "Google Calendar Tools"
	case "meet":
		return "Google Meet Tools"
	case "tasks":
		return "Google Tasks Tools"
	default:
		return "Other"
	}
}

func generateToolMarkdown(tool mcp.Tool) string {
	var sb strings.Builder

	// Tool name
	sb.WriteString(fmt.Sprintf("### %s\n\n", tool.Name))

	// Description
	if tool.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", tool.Description))
	}

	// Input schema
	if tool.InputSchema.Properties != nil && len(tool.InputSchema.Properties) > 0 {
		sb.WriteString("**Arguments:**\n")

		// Sort properties for consistent output
		propNames := make([]string, 0, len(tool.InputSchema.Properties))
		for name := range tool.InputSchema.Properties {
			propNames = append(propNames, name)
		}
		sort.Strings(propNames)

		for _, name := range propNames {
			prop := tool.InputSchema.Properties[name]
			isRequired := contains(tool.InputSchema.Required, name)

			requiredStr := "optional"
			if isRequired {
				requiredStr = "required"
			}

			// Get property type and description from the property map
			propMap, ok := prop.(map[string]interface{})
			if !ok {
				continue
			}

			propType := getPropertyType(propMap)

			sb.WriteString(fmt.Sprintf("- `%s` (%s): ", name, requiredStr))

			// Get description
			if desc, ok := propMap["description"].(string); ok {
				sb.WriteString(desc)
			} else {
				sb.WriteString(fmt.Sprintf("%s parameter", propType))
			}

			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func getPropertyType(prop map[string]interface{}) string {
	if t, ok := prop["type"].(string); ok {
		return t
	}
	return "any"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
