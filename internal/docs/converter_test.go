package docs

import (
	"strings"
	"testing"

	docs "google.golang.org/api/docs/v1"
)

func TestDocumentToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		doc      *docs.Document
		expected string
		wantErr  bool
	}{
		{
			name:    "Nil document",
			doc:     nil,
			wantErr: true,
		},
		{
			name: "Simple document with title",
			doc: &docs.Document{
				Title: "Test Document",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "This is a test.\n",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Test Document\n\nThis is a test.\n\n\n",
		},
		{
			name: "Document with headings",
			doc: &docs.Document{
				Title: "Document",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								ParagraphStyle: &docs.ParagraphStyle{
									NamedStyleType: "HEADING_1",
								},
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Heading 1\n",
										},
									},
								},
							},
						},
						{
							Paragraph: &docs.Paragraph{
								ParagraphStyle: &docs.ParagraphStyle{
									NamedStyleType: "HEADING_2",
								},
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Heading 2\n",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Document\n\n# Heading 1\n\n\n## Heading 2\n\n\n",
		},
		{
			name: "Document with bold and italic",
			doc: &docs.Document{
				Title: "Formatted Text",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Bold text",
											TextStyle: &docs.TextStyle{
												Bold: true,
											},
										},
									},
								},
							},
						},
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Italic text",
											TextStyle: &docs.TextStyle{
												Italic: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Formatted Text\n\n**Bold text**\n\n*Italic text*\n\n",
		},
		{
			name: "Document with bullet list",
			doc: &docs.Document{
				Title: "List Document",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Bullet: &docs.Bullet{
									ListId: "list1",
								},
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Item 1\n",
										},
									},
								},
							},
						},
						{
							Paragraph: &docs.Paragraph{
								Bullet: &docs.Bullet{
									ListId: "list1",
								},
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Item 2\n",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# List Document\n\n- Item 1\n\n- Item 2\n\n",
		},
		{
			name: "Document with link",
			doc: &docs.Document{
				Title: "Link Document",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Click here",
											TextStyle: &docs.TextStyle{
												Link: &docs.Link{
													Url: "https://example.com",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Link Document\n\n[Click here](https://example.com)\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DocumentToMarkdown(tt.doc)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DocumentToMarkdown() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("DocumentToMarkdown() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("DocumentToMarkdown() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestDocumentToPlainText(t *testing.T) {
	tests := []struct {
		name     string
		doc      *docs.Document
		expected string
		wantErr  bool
	}{
		{
			name:    "Nil document",
			doc:     nil,
			wantErr: true,
		},
		{
			name: "Simple document",
			doc: &docs.Document{
				Title: "Test Document",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "This is plain text.\n",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "Test Document\n\nThis is plain text.\n",
		},
		{
			name: "Document with multiple paragraphs",
			doc: &docs.Document{
				Title: "Multi Paragraph",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "First paragraph.\n",
										},
									},
								},
							},
						},
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Second paragraph.\n",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "Multi Paragraph\n\nFirst paragraph.\nSecond paragraph.\n",
		},
		{
			name: "Document with formatted text (should strip formatting)",
			doc: &docs.Document{
				Title: "Formatted",
				Body: &docs.Body{
					Content: []*docs.StructuralElement{
						{
							Paragraph: &docs.Paragraph{
								Elements: []*docs.ParagraphElement{
									{
										TextRun: &docs.TextRun{
											Content: "Bold text",
											TextStyle: &docs.TextStyle{
												Bold: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "Formatted\n\nBold text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DocumentToPlainText(tt.doc)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DocumentToPlainText() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("DocumentToPlainText() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("DocumentToPlainText() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestDocumentToMarkdown_WithTabs(t *testing.T) {
	tests := []struct {
		name     string
		doc      *docs.Document
		expected string
		wantErr  bool
	}{
		{
			name: "Document with single tab",
			doc: &docs.Document{
				Title: "Tabbed Document",
				Tabs: []*docs.Tab{
					{
						TabProperties: &docs.TabProperties{
							Title: "First Tab",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "Content in first tab.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Tabbed Document\n\n## Tab: First Tab\n\nContent in first tab.\n\n\n",
		},
		{
			name: "Document with multiple tabs",
			doc: &docs.Document{
				Title: "Multi-Tab Document",
				Tabs: []*docs.Tab{
					{
						TabProperties: &docs.TabProperties{
							Title: "Tab One",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "First tab content.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					{
						TabProperties: &docs.TabProperties{
							Title: "Tab Two",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "Second tab content.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Multi-Tab Document\n\n## Tab: Tab One\n\nFirst tab content.\n\n\n## Tab: Tab Two\n\nSecond tab content.\n\n\n",
		},
		{
			name: "Document with child tabs",
			doc: &docs.Document{
				Title: "Nested Tabs Document",
				Tabs: []*docs.Tab{
					{
						TabProperties: &docs.TabProperties{
							Title: "Parent Tab",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "Parent content.\n",
													},
												},
											},
										},
									},
								},
							},
						},
						ChildTabs: []*docs.Tab{
							{
								TabProperties: &docs.TabProperties{
									Title: "Child Tab",
								},
								DocumentTab: &docs.DocumentTab{
									Body: &docs.Body{
										Content: []*docs.StructuralElement{
											{
												Paragraph: &docs.Paragraph{
													Elements: []*docs.ParagraphElement{
														{
															TextRun: &docs.TextRun{
																Content: "Child content.\n",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Nested Tabs Document\n\n## Tab: Parent Tab\n\nParent content.\n\n\n### Child Tab\n\nChild content.\n\n\n",
		},
		{
			name: "Document with tab without title",
			doc: &docs.Document{
				Title: "Untitled Tab Document",
				Tabs: []*docs.Tab{
					{
						TabProperties: &docs.TabProperties{
							Title: "First Tab",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "First.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					{
						TabProperties: &docs.TabProperties{},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "Second.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "# Untitled Tab Document\n\n## Tab: First Tab\n\nFirst.\n\n\n## Tab 2\n\nSecond.\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DocumentToMarkdown(tt.doc)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DocumentToMarkdown() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("DocumentToMarkdown() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("DocumentToMarkdown() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestDocumentToPlainText_WithTabs(t *testing.T) {
	tests := []struct {
		name     string
		doc      *docs.Document
		expected string
		wantErr  bool
	}{
		{
			name: "Tabbed document with single tab",
			doc: &docs.Document{
				Title: "Tabbed Document",
				Tabs: []*docs.Tab{
					{
						TabProperties: &docs.TabProperties{
							Title: "First Tab",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "Content in first tab.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "Tabbed Document\n\n=== First Tab ===\n\nContent in first tab.\n\n",
		},
		{
			name: "Tabbed document with multiple tabs",
			doc: &docs.Document{
				Title: "Multi-Tab Document",
				Tabs: []*docs.Tab{
					{
						TabProperties: &docs.TabProperties{
							Title: "Tab One",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "First tab content.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					{
						TabProperties: &docs.TabProperties{
							Title: "Tab Two",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "Second tab content.\n",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "Multi-Tab Document\n\n=== Tab One ===\n\nFirst tab content.\n\n=== Tab Two ===\n\nSecond tab content.\n\n",
		},
		{
			name: "Tabbed document with child tabs",
			doc: &docs.Document{
				Title: "Nested Tabs",
				Tabs: []*docs.Tab{
					{
						TabProperties: &docs.TabProperties{
							Title: "Parent",
						},
						DocumentTab: &docs.DocumentTab{
							Body: &docs.Body{
								Content: []*docs.StructuralElement{
									{
										Paragraph: &docs.Paragraph{
											Elements: []*docs.ParagraphElement{
												{
													TextRun: &docs.TextRun{
														Content: "Parent content.\n",
													},
												},
											},
										},
									},
								},
							},
						},
						ChildTabs: []*docs.Tab{
							{
								TabProperties: &docs.TabProperties{
									Title: "Child",
								},
								DocumentTab: &docs.DocumentTab{
									Body: &docs.Body{
										Content: []*docs.StructuralElement{
											{
												Paragraph: &docs.Paragraph{
													Elements: []*docs.ParagraphElement{
														{
															TextRun: &docs.TextRun{
																Content: "Child content.\n",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "Nested Tabs\n\n=== Parent ===\n\nParent content.\n  --- Child ---\n\nChild content.\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DocumentToPlainText(tt.doc)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DocumentToPlainText() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("DocumentToPlainText() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("DocumentToPlainText() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestProcessTable(t *testing.T) {
	table := &docs.Table{
		TableRows: []*docs.TableRow{
			{
				TableCells: []*docs.TableCell{
					{
						Content: []*docs.StructuralElement{
							{
								Paragraph: &docs.Paragraph{
									Elements: []*docs.ParagraphElement{
										{
											TextRun: &docs.TextRun{
												Content: "Header 1\n",
											},
										},
									},
								},
							},
						},
					},
					{
						Content: []*docs.StructuralElement{
							{
								Paragraph: &docs.Paragraph{
									Elements: []*docs.ParagraphElement{
										{
											TextRun: &docs.TextRun{
												Content: "Header 2\n",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				TableCells: []*docs.TableCell{
					{
						Content: []*docs.StructuralElement{
							{
								Paragraph: &docs.Paragraph{
									Elements: []*docs.ParagraphElement{
										{
											TextRun: &docs.TextRun{
												Content: "Cell 1\n",
											},
										},
									},
								},
							},
						},
					},
					{
						Content: []*docs.StructuralElement{
							{
								Paragraph: &docs.Paragraph{
									Elements: []*docs.ParagraphElement{
										{
											TextRun: &docs.TextRun{
												Content: "Cell 2\n",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var md strings.Builder
	processTable(&md, table)
	result := md.String()

	// Check that it contains table markdown elements
	if !strings.Contains(result, "|") {
		t.Errorf("processTable() result should contain table pipes")
	}
	if !strings.Contains(result, "---") {
		t.Errorf("processTable() result should contain header separator")
	}
	if !strings.Contains(result, "Header 1") || !strings.Contains(result, "Header 2") {
		t.Errorf("processTable() result should contain header text")
	}
	if !strings.Contains(result, "Cell 1") || !strings.Contains(result, "Cell 2") {
		t.Errorf("processTable() result should contain cell text")
	}
}
