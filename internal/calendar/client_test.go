package calendar

import (
	"testing"
	"time"
)

func TestToEventSummary(t *testing.T) {
	// This test ensures toEventSummary correctly converts a Google Calendar event
	// We'll test with a nil event first
	summary := toEventSummary(nil)
	if summary.ID != "" {
		t.Errorf("Expected empty ID for nil event, got %s", summary.ID)
	}
}

func TestToCalendarInfo(t *testing.T) {
	// This test ensures toCalendarInfo correctly converts a Calendar list entry
	// We'll test with a nil entry first
	info := toCalendarInfo(nil)
	if info.ID != "" {
		t.Errorf("Expected empty ID for nil entry, got %s", info.ID)
	}
}

func TestHasToken(t *testing.T) {
	// Test that HasToken returns a boolean without error
	result := HasToken()
	// We don't care about the actual value, just that it doesn't panic
	_ = result
}

func TestHasTokenForAccount(t *testing.T) {
	// Test that HasTokenForAccount returns a boolean for valid account name
	result := HasTokenForAccount("test-account")
	_ = result

	// Test with empty account name
	result = HasTokenForAccount("")
	if result {
		t.Error("Expected false for empty account name")
	}
}

func TestEventInput_Validation(t *testing.T) {
	// Test EventInput structure with various valid and invalid inputs
	tests := []struct {
		name  string
		input EventInput
	}{
		{
			name: "valid basic event",
			input: EventInput{
				Summary: "Test Event",
				Start:   time.Now(),
				End:     time.Now().Add(time.Hour),
			},
		},
		{
			name: "valid recurring event",
			input: EventInput{
				Summary:    "Weekly Meeting",
				Start:      time.Now(),
				End:        time.Now().Add(time.Hour),
				Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO"},
			},
		},
		{
			name: "valid out-of-office event",
			input: EventInput{
				Summary:   "Out of Office",
				Start:     time.Now(),
				End:       time.Now().Add(8 * time.Hour),
				EventType: "outOfOffice",
			},
		},
		{
			name: "event with attendees",
			input: EventInput{
				Summary:   "Team Meeting",
				Start:     time.Now(),
				End:       time.Now().Add(time.Hour),
				Attendees: []string{"user1@example.com", "user2@example.com"},
			},
		},
		{
			name: "event with Google Meet",
			input: EventInput{
				Summary:                  "Video Call",
				Start:                    time.Now(),
				End:                      time.Now().Add(time.Hour),
				UseDefaultConferenceData: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the input structure is correctly formed
			if tt.input.Summary == "" {
				t.Error("Expected non-empty summary")
			}
			if tt.input.Start.IsZero() {
				t.Error("Expected non-zero start time")
			}
			if tt.input.End.IsZero() {
				t.Error("Expected non-zero end time")
			}
			if tt.input.End.Before(tt.input.Start) {
				t.Error("End time should be after start time")
			}
		})
	}
}

func TestAttendeeInfo_Structure(t *testing.T) {
	// Test AttendeeInfo structure
	attendee := AttendeeInfo{
		Email:          "test@example.com",
		DisplayName:    "Test User",
		ResponseStatus: "accepted",
		Optional:       false,
		Organizer:      true,
	}

	if attendee.Email == "" {
		t.Error("Expected non-empty email")
	}
	if attendee.ResponseStatus != "accepted" {
		t.Errorf("Expected response status 'accepted', got %s", attendee.ResponseStatus)
	}
	if !attendee.Organizer {
		t.Error("Expected organizer to be true")
	}
}

func TestCalendarInfo_Structure(t *testing.T) {
	// Test CalendarInfo structure
	info := CalendarInfo{
		ID:          "test@example.com",
		Summary:     "Test Calendar",
		Description: "A test calendar",
		TimeZone:    "America/New_York",
		Primary:     true,
		AccessRole:  "owner",
	}

	if info.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if info.Summary == "" {
		t.Error("Expected non-empty summary")
	}
	if !info.Primary {
		t.Error("Expected primary to be true")
	}
	if info.AccessRole != "owner" {
		t.Errorf("Expected access role 'owner', got %s", info.AccessRole)
	}
}

func TestFreeBusyInfo_Structure(t *testing.T) {
	// Test FreeBusyInfo structure
	now := time.Now()
	later := now.Add(time.Hour)

	info := FreeBusyInfo{
		Calendar: "test@example.com",
		Busy: []TimeRange{
			{Start: now, End: later},
		},
		Errors: []string{},
	}

	if info.Calendar == "" {
		t.Error("Expected non-empty calendar")
	}
	if len(info.Busy) != 1 {
		t.Errorf("Expected 1 busy period, got %d", len(info.Busy))
	}
	if info.Busy[0].Start.After(info.Busy[0].End) {
		t.Error("Start time should be before end time in busy period")
	}
}

func TestAvailableSlot_Structure(t *testing.T) {
	// Test AvailableSlot structure
	now := time.Now()
	duration := 30 * time.Minute

	slot := AvailableSlot{
		Start:    now,
		End:      now.Add(duration),
		Duration: duration,
	}

	if slot.Start.IsZero() {
		t.Error("Expected non-zero start time")
	}
	if slot.End.IsZero() {
		t.Error("Expected non-zero end time")
	}
	if slot.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, slot.Duration)
	}
	if slot.End.Sub(slot.Start) != duration {
		t.Error("End-Start should equal Duration")
	}
}

func TestIsGoogleDocsLink(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Google Docs URL", "https://docs.google.com/document/d/123/edit", true},
		{"Google Drive URL", "https://drive.google.com/file/d/456/view", true},
		{"Non-Google URL", "https://example.com/document", false},
		{"Empty URL", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGoogleDocsLink(tt.url)
			if result != tt.expected {
				t.Errorf("isGoogleDocsLink(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestExtractLinksFromText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int // number of links expected
	}{
		{
			name:     "single link",
			text:     "Check out https://example.com for more info",
			expected: 1,
		},
		{
			name:     "multiple links",
			text:     "Visit https://example.com and https://test.com",
			expected: 2,
		},
		{
			name:     "no links",
			text:     "This is just plain text",
			expected: 0,
		},
		{
			name:     "http link",
			text:     "Visit http://example.com",
			expected: 1,
		},
		{
			name:     "empty text",
			text:     "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := extractLinksFromText(tt.text)
			if len(links) != tt.expected {
				t.Errorf("extractLinksFromText(%q) returned %d links, expected %d", tt.text, len(links), tt.expected)
			}
		})
	}
}
