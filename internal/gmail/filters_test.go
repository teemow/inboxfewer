package gmail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	gmail "google.golang.org/api/gmail/v1"
)

func TestConvertGmailFilterToFilterInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    *gmail.Filter
		expected *FilterInfo
	}{
		{
			name: "basic filter with from and archive",
			input: &gmail.Filter{
				Id: "filter123",
				Criteria: &gmail.FilterCriteria{
					From: "spam@example.com",
				},
				Action: &gmail.FilterAction{
					RemoveLabelIds: []string{"INBOX"},
				},
			},
			expected: &FilterInfo{
				ID: "filter123",
				Criteria: FilterCriteria{
					From: "spam@example.com",
				},
				Action: FilterAction{
					Archive:        true,
					RemoveLabelIDs: []string{"INBOX"},
				},
			},
		},
		{
			name: "filter with subject and add label",
			input: &gmail.Filter{
				Id: "filter456",
				Criteria: &gmail.FilterCriteria{
					Subject: "Important",
				},
				Action: &gmail.FilterAction{
					AddLabelIds: []string{"Label_1"},
				},
			},
			expected: &FilterInfo{
				ID: "filter456",
				Criteria: FilterCriteria{
					Subject: "Important",
				},
				Action: FilterAction{
					AddLabelIDs: []string{"Label_1"},
				},
			},
		},
		{
			name: "filter with mark as read",
			input: &gmail.Filter{
				Id: "filter789",
				Criteria: &gmail.FilterCriteria{
					From: "newsletter@example.com",
				},
				Action: &gmail.FilterAction{
					RemoveLabelIds: []string{"UNREAD"},
				},
			},
			expected: &FilterInfo{
				ID: "filter789",
				Criteria: FilterCriteria{
					From: "newsletter@example.com",
				},
				Action: FilterAction{
					MarkAsRead:     true,
					RemoveLabelIDs: []string{"UNREAD"},
				},
			},
		},
		{
			name: "filter with star",
			input: &gmail.Filter{
				Id: "filter101",
				Criteria: &gmail.FilterCriteria{
					To: "important@example.com",
				},
				Action: &gmail.FilterAction{
					AddLabelIds: []string{"STARRED"},
				},
			},
			expected: &FilterInfo{
				ID: "filter101",
				Criteria: FilterCriteria{
					To: "important@example.com",
				},
				Action: FilterAction{
					Star:        true,
					AddLabelIDs: []string{"STARRED"},
				},
			},
		},
		{
			name: "filter with delete (trash)",
			input: &gmail.Filter{
				Id: "filter202",
				Criteria: &gmail.FilterCriteria{
					From: "spam@example.com",
				},
				Action: &gmail.FilterAction{
					AddLabelIds: []string{"TRASH"},
				},
			},
			expected: &FilterInfo{
				ID: "filter202",
				Criteria: FilterCriteria{
					From: "spam@example.com",
				},
				Action: FilterAction{
					Delete:      true,
					AddLabelIDs: []string{"TRASH"},
				},
			},
		},
		{
			name: "complex filter with multiple actions",
			input: &gmail.Filter{
				Id: "filter303",
				Criteria: &gmail.FilterCriteria{
					From:          "important@example.com",
					HasAttachment: true,
				},
				Action: &gmail.FilterAction{
					AddLabelIds:    []string{"STARRED", "Label_1"},
					RemoveLabelIds: []string{"UNREAD"},
				},
			},
			expected: &FilterInfo{
				ID: "filter303",
				Criteria: FilterCriteria{
					From:          "important@example.com",
					HasAttachment: true,
				},
				Action: FilterAction{
					Star:           true,
					MarkAsRead:     true,
					AddLabelIDs:    []string{"STARRED", "Label_1"},
					RemoveLabelIDs: []string{"UNREAD"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGmailFilterToFilterInfo(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterCriteriaAndAction(t *testing.T) {
	// Test that FilterCriteria and FilterAction are properly structured
	criteria := FilterCriteria{
		From:           "test@example.com",
		To:             "recipient@example.com",
		Subject:        "Test Subject",
		Query:          "has:attachment",
		HasAttachment:  true,
		Size:           1000000,
		SizeComparison: "larger",
	}

	assert.Equal(t, "test@example.com", criteria.From)
	assert.Equal(t, "recipient@example.com", criteria.To)
	assert.Equal(t, "Test Subject", criteria.Subject)
	assert.True(t, criteria.HasAttachment)

	action := FilterAction{
		AddLabelIDs:    []string{"Label_1", "Label_2"},
		RemoveLabelIDs: []string{"INBOX"},
		Forward:        "forward@example.com",
		Archive:        true,
		MarkAsRead:     true,
		Star:           true,
		MarkAsSpam:     false,
		Delete:         false,
	}

	assert.Len(t, action.AddLabelIDs, 2)
	assert.True(t, action.Archive)
	assert.True(t, action.MarkAsRead)
	assert.True(t, action.Star)
}
