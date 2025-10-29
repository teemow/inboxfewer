package docs

import (
	"fmt"
	"strings"

	docs "google.golang.org/api/docs/v1"
)

// DocumentToMarkdown converts a Google Doc to Markdown format
func DocumentToMarkdown(doc *docs.Document) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("document is nil")
	}

	var md strings.Builder

	// Add title as H1
	if doc.Title != "" {
		md.WriteString("# ")
		md.WriteString(doc.Title)
		md.WriteString("\n\n")
	}

	// Process document body
	if doc.Body != nil && doc.Body.Content != nil {
		for _, element := range doc.Body.Content {
			processStructuralElement(&md, element)
		}
	}

	return md.String(), nil
}

// DocumentToPlainText extracts plain text from a Google Doc
func DocumentToPlainText(doc *docs.Document) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("document is nil")
	}

	var text strings.Builder

	// Add title
	if doc.Title != "" {
		text.WriteString(doc.Title)
		text.WriteString("\n\n")
	}

	// Process document body
	if doc.Body != nil && doc.Body.Content != nil {
		for _, element := range doc.Body.Content {
			extractPlainText(&text, element)
		}
	}

	return text.String(), nil
}

// processStructuralElement processes a structural element and converts it to Markdown
func processStructuralElement(md *strings.Builder, element *docs.StructuralElement) {
	if element.Paragraph != nil {
		processParagraph(md, element.Paragraph)
	} else if element.Table != nil {
		processTable(md, element.Table)
	} else if element.SectionBreak != nil {
		// Section breaks don't produce visible content
		md.WriteString("\n---\n\n")
	}
}

// processParagraph processes a paragraph and converts it to Markdown
func processParagraph(md *strings.Builder, para *docs.Paragraph) {
	if para == nil || para.Elements == nil {
		return
	}

	// Determine the heading level or list style
	headingLevel := 0
	listStyle := ""

	if para.ParagraphStyle != nil {
		if para.ParagraphStyle.NamedStyleType != "" {
			switch para.ParagraphStyle.NamedStyleType {
			case "HEADING_1":
				headingLevel = 1
			case "HEADING_2":
				headingLevel = 2
			case "HEADING_3":
				headingLevel = 3
			case "HEADING_4":
				headingLevel = 4
			case "HEADING_5":
				headingLevel = 5
			case "HEADING_6":
				headingLevel = 6
			}
		}
	}

	// Check for bullet or numbered list
	if para.Bullet != nil {
		listStyle = "bullet"
		// For simplicity, we'll treat all lists as bullet lists
		// A more robust implementation would need to track list types by ListId
	}

	// Write heading prefix
	if headingLevel > 0 {
		md.WriteString(strings.Repeat("#", headingLevel))
		md.WriteString(" ")
	}

	// Write list prefix
	if listStyle == "bullet" {
		md.WriteString("- ")
	}

	// Process paragraph elements
	for _, elem := range para.Elements {
		if elem.TextRun != nil {
			processTextRun(md, elem.TextRun)
		} else if elem.InlineObjectElement != nil {
			// Handle inline objects (images, etc.)
			md.WriteString("[inline object]")
		}
	}

	// Add line break
	md.WriteString("\n")

	// Add extra line break after headings and normal paragraphs
	if headingLevel > 0 || listStyle == "" {
		md.WriteString("\n")
	}
}

// processTextRun processes a text run and applies Markdown formatting
func processTextRun(md *strings.Builder, textRun *docs.TextRun) {
	if textRun.Content == "" {
		return
	}

	content := textRun.Content
	style := textRun.TextStyle

	if style != nil {
		// Apply formatting
		isBold := style.Bold
		isItalic := style.Italic
		isCode := style.WeightedFontFamily != nil && strings.Contains(style.WeightedFontFamily.FontFamily, "Courier")

		// Check for links
		if style.Link != nil && style.Link.Url != "" {
			// Format as markdown link
			linkText := strings.TrimSpace(content)
			md.WriteString("[")
			md.WriteString(linkText)
			md.WriteString("](")
			md.WriteString(style.Link.Url)
			md.WriteString(")")
			return
		}

		// Apply code formatting
		if isCode {
			md.WriteString("`")
			md.WriteString(strings.TrimSpace(content))
			md.WriteString("`")
			return
		}

		// Apply bold/italic formatting
		if isBold && isItalic {
			md.WriteString("***")
			md.WriteString(content)
			md.WriteString("***")
		} else if isBold {
			md.WriteString("**")
			md.WriteString(content)
			md.WriteString("**")
		} else if isItalic {
			md.WriteString("*")
			md.WriteString(content)
			md.WriteString("*")
		} else {
			md.WriteString(content)
		}
	} else {
		md.WriteString(content)
	}
}

// processTable processes a table and converts it to Markdown table format
func processTable(md *strings.Builder, table *docs.Table) {
	if table == nil || table.TableRows == nil || len(table.TableRows) == 0 {
		return
	}

	// Process each row
	for rowIndex, row := range table.TableRows {
		if row.TableCells == nil {
			continue
		}

		md.WriteString("|")
		for _, cell := range row.TableCells {
			md.WriteString(" ")
			if cell.Content != nil {
				for _, element := range cell.Content {
					if element.Paragraph != nil {
						// Extract text from paragraph without formatting for table cells
						for _, elem := range element.Paragraph.Elements {
							if elem.TextRun != nil {
								// Simple text extraction for table cells
								content := strings.TrimSpace(elem.TextRun.Content)
								content = strings.ReplaceAll(content, "\n", " ")
								md.WriteString(content)
							}
						}
					}
				}
			}
			md.WriteString(" |")
		}
		md.WriteString("\n")

		// Add header separator after first row
		if rowIndex == 0 {
			md.WriteString("|")
			for range row.TableCells {
				md.WriteString(" --- |")
			}
			md.WriteString("\n")
		}
	}

	md.WriteString("\n")
}

// extractPlainText extracts plain text from a structural element
func extractPlainText(text *strings.Builder, element *docs.StructuralElement) {
	if element.Paragraph != nil {
		extractParagraphText(text, element.Paragraph)
	} else if element.Table != nil {
		extractTableText(text, element.Table)
	}
}

// extractParagraphText extracts plain text from a paragraph
func extractParagraphText(text *strings.Builder, para *docs.Paragraph) {
	if para == nil || para.Elements == nil {
		return
	}

	for _, elem := range para.Elements {
		if elem.TextRun != nil && elem.TextRun.Content != "" {
			text.WriteString(elem.TextRun.Content)
		}
	}
}

// extractTableText extracts plain text from a table
func extractTableText(text *strings.Builder, table *docs.Table) {
	if table == nil || table.TableRows == nil {
		return
	}

	for _, row := range table.TableRows {
		if row.TableCells == nil {
			continue
		}

		for _, cell := range row.TableCells {
			if cell.Content != nil {
				for _, element := range cell.Content {
					if element.Paragraph != nil {
						extractParagraphText(text, element.Paragraph)
					}
				}
			}
			text.WriteString("\t")
		}
		text.WriteString("\n")
	}
}
