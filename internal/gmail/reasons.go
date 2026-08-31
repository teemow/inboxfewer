package gmail

import (
	"strings"

	gmail "google.golang.org/api/gmail/v1"
)

// githubReasonHeader is set by GitHub on every notification email to explain why
// the recipient received it. Known values include "mention", "team_mention",
// "review_requested", "assign", "author", "comment", "subscribed", "push",
// "manual", "state_change" and "ci_activity".
const githubReasonHeader = "X-GitHub-Reason"

// DefaultPersonalReasons are the notification reasons that mean the recipient was
// addressed personally: mentioned by handle, assigned, or the author. Reasons such
// as "team_mention" and "review_requested" are excluded, because they are usually
// sent to a whole team.
var DefaultPersonalReasons = []string{"mention", "assign", "author"}

// headerValueFold returns the first header whose name matches case-insensitively.
//
// HeaderValue in classifier.go compares names exactly. That is fine for the
// Message-ID lookups it was written for, but here a missed header would fail open
// and archive a thread that should have been kept, so match loosely instead.
func headerValueFold(m *gmail.Message, name string) string {
	if m == nil || m.Payload == nil {
		return ""
	}
	for _, h := range m.Payload.Headers {
		if h != nil && strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// ThreadReasons returns the distinct X-GitHub-Reason values found across the
// messages of a thread, lowercased and in the order they first appear. Messages
// without the header are ignored.
//
// PopulateThread fetches threads with format "full", so the header is already
// present and no extra Gmail API call is needed.
func ThreadReasons(t *gmail.Thread) []string {
	if t == nil {
		return nil
	}
	seen := make(map[string]bool)
	var reasons []string
	for _, m := range t.Messages {
		reason := strings.ToLower(strings.TrimSpace(headerValueFold(m, githubReasonHeader)))
		if reason == "" || seen[reason] {
			continue
		}
		seen[reason] = true
		reasons = append(reasons, reason)
	}
	return reasons
}

// IsPersonallyAddressed reports whether any message in the thread carries one of
// the given reasons. A thread that notifies the recipient personally even once is
// treated as personal as a whole.
func IsPersonallyAddressed(t *gmail.Thread, reasons []string) bool {
	if len(reasons) == 0 {
		return false
	}
	wanted := make(map[string]bool, len(reasons))
	for _, r := range reasons {
		wanted[strings.ToLower(strings.TrimSpace(r))] = true
	}
	for _, reason := range ThreadReasons(t) {
		if wanted[reason] {
			return true
		}
	}
	return false
}
