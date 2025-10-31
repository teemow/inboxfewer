package gmail

import (
	"fmt"

	gmail "google.golang.org/api/gmail/v1"
)

// FilterCriteria represents the criteria for a Gmail filter
type FilterCriteria struct {
	From           string // Email addresses to filter from
	To             string // Email addresses to filter to
	Subject        string // Words in the subject line
	Query          string // Gmail search query
	HasAttachment  bool   // Whether the message has attachments
	Size           int64  // Message size in bytes (use with SizeComparison)
	SizeComparison string // "larger" or "smaller"
}

// FilterAction represents the actions to take when a filter matches
type FilterAction struct {
	AddLabelIDs    []string // Label IDs to add
	RemoveLabelIDs []string // Label IDs to remove
	Forward        string   // Email address to forward to
	Archive        bool     // Remove from inbox (remove INBOX label)
	MarkAsRead     bool     // Mark as read
	Star           bool     // Add star
	MarkAsSpam     bool     // Mark as spam
	Delete         bool     // Send to trash
}

// FilterInfo represents a Gmail filter with its criteria and actions
type FilterInfo struct {
	ID       string
	Criteria FilterCriteria
	Action   FilterAction
}

// CreateFilter creates a new Gmail filter
func (c *Client) CreateFilter(criteria FilterCriteria, action FilterAction) (*FilterInfo, error) {
	// Build Gmail filter criteria
	gmailCriteria := &gmail.FilterCriteria{}

	if criteria.From != "" {
		gmailCriteria.From = criteria.From
	}
	if criteria.To != "" {
		gmailCriteria.To = criteria.To
	}
	if criteria.Subject != "" {
		gmailCriteria.Subject = criteria.Subject
	}
	if criteria.Query != "" {
		gmailCriteria.Query = criteria.Query
	}
	if criteria.HasAttachment {
		gmailCriteria.HasAttachment = true
	}
	if criteria.Size > 0 {
		gmailCriteria.Size = criteria.Size
		if criteria.SizeComparison != "" {
			gmailCriteria.SizeComparison = criteria.SizeComparison
		}
	}

	// Build Gmail filter action
	gmailAction := &gmail.FilterAction{}

	if len(action.AddLabelIDs) > 0 {
		gmailAction.AddLabelIds = action.AddLabelIDs
	}
	if len(action.RemoveLabelIDs) > 0 {
		gmailAction.RemoveLabelIds = action.RemoveLabelIDs
	}
	if action.Forward != "" {
		gmailAction.Forward = action.Forward
	}

	// Archive means removing INBOX label
	if action.Archive {
		if gmailAction.RemoveLabelIds == nil {
			gmailAction.RemoveLabelIds = []string{}
		}
		gmailAction.RemoveLabelIds = append(gmailAction.RemoveLabelIds, "INBOX")
	}

	if action.MarkAsRead {
		if gmailAction.RemoveLabelIds == nil {
			gmailAction.RemoveLabelIds = []string{}
		}
		gmailAction.RemoveLabelIds = append(gmailAction.RemoveLabelIds, "UNREAD")
	}

	if action.Star {
		if gmailAction.AddLabelIds == nil {
			gmailAction.AddLabelIds = []string{}
		}
		gmailAction.AddLabelIds = append(gmailAction.AddLabelIds, "STARRED")
	}

	if action.MarkAsSpam {
		if gmailAction.AddLabelIds == nil {
			gmailAction.AddLabelIds = []string{}
		}
		gmailAction.AddLabelIds = append(gmailAction.AddLabelIds, "SPAM")
	}

	if action.Delete {
		if gmailAction.AddLabelIds == nil {
			gmailAction.AddLabelIds = []string{}
		}
		gmailAction.AddLabelIds = append(gmailAction.AddLabelIds, "TRASH")
	}

	// Create the filter
	filter := &gmail.Filter{
		Criteria: gmailCriteria,
		Action:   gmailAction,
	}

	created, err := c.svc.Settings.Filters.Create("me", filter).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create filter: %w", err)
	}

	// Convert to FilterInfo
	return convertGmailFilterToFilterInfo(created), nil
}

// ListFilters lists all Gmail filters for the user
func (c *Client) ListFilters() ([]*FilterInfo, error) {
	resp, err := c.svc.Settings.Filters.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list filters: %w", err)
	}

	filters := make([]*FilterInfo, 0, len(resp.Filter))
	for _, f := range resp.Filter {
		filters = append(filters, convertGmailFilterToFilterInfo(f))
	}

	return filters, nil
}

// GetFilter retrieves a specific filter by ID
func (c *Client) GetFilter(filterID string) (*FilterInfo, error) {
	filter, err := c.svc.Settings.Filters.Get("me", filterID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get filter: %w", err)
	}

	return convertGmailFilterToFilterInfo(filter), nil
}

// DeleteFilter deletes a filter by ID
func (c *Client) DeleteFilter(filterID string) error {
	err := c.svc.Settings.Filters.Delete("me", filterID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete filter: %w", err)
	}
	return nil
}

// convertGmailFilterToFilterInfo converts a Gmail API filter to FilterInfo
func convertGmailFilterToFilterInfo(f *gmail.Filter) *FilterInfo {
	info := &FilterInfo{
		ID: f.Id,
	}

	// Convert criteria
	if f.Criteria != nil {
		info.Criteria = FilterCriteria{
			From:           f.Criteria.From,
			To:             f.Criteria.To,
			Subject:        f.Criteria.Subject,
			Query:          f.Criteria.Query,
			HasAttachment:  f.Criteria.HasAttachment,
			Size:           f.Criteria.Size,
			SizeComparison: f.Criteria.SizeComparison,
		}
	}

	// Convert action
	if f.Action != nil {
		info.Action = FilterAction{
			AddLabelIDs:    f.Action.AddLabelIds,
			RemoveLabelIDs: f.Action.RemoveLabelIds,
			Forward:        f.Action.Forward,
		}

		// Check for special actions
		for _, labelID := range f.Action.RemoveLabelIds {
			if labelID == "INBOX" {
				info.Action.Archive = true
			}
			if labelID == "UNREAD" {
				info.Action.MarkAsRead = true
			}
		}

		for _, labelID := range f.Action.AddLabelIds {
			if labelID == "STARRED" {
				info.Action.Star = true
			}
			if labelID == "SPAM" {
				info.Action.MarkAsSpam = true
			}
			if labelID == "TRASH" {
				info.Action.Delete = true
			}
		}
	}

	return info
}

// ListLabels lists all Gmail labels for the user
// This is useful for getting label IDs to use in filters
func (c *Client) ListLabels() ([]*gmail.Label, error) {
	resp, err := c.svc.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	return resp.Labels, nil
}
