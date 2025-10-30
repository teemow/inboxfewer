package calendar

import (
	"time"

	calendar "google.golang.org/api/calendar/v3"
)

// EventInput represents the input for creating or updating a calendar event
type EventInput struct {
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	TimeZone    string
	Attendees   []string
	Recurrence  []string // RRULE, EXRULE, RDATE, EXDATE

	// Event type: "default", "outOfOffice", "focusTime", "workingLocation"
	EventType string

	// Guest permissions
	GuestsCanModify         bool
	GuestsCanInviteOthers   bool
	GuestsCanSeeOtherGuests bool

	// Conference data
	UseDefaultConferenceData bool // Automatically add Google Meet
}

// EventSummary represents a simplified calendar event for listing
type EventSummary struct {
	ID          string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	Creator     string
	Organizer   string
	Status      string
	Attendees   []AttendeeInfo
	MeetLink    string
	EventType   string
}

// AttendeeInfo represents information about an event attendee
type AttendeeInfo struct {
	Email          string
	DisplayName    string
	ResponseStatus string // "needsAction", "declined", "tentative", "accepted"
	Optional       bool
	Organizer      bool
}

// CalendarInfo represents information about a calendar
type CalendarInfo struct {
	ID          string
	Summary     string
	Description string
	TimeZone    string
	Primary     bool
	AccessRole  string // "owner", "writer", "reader", "freeBusyReader"
}

// FreeBusyInfo represents availability information for a calendar
type FreeBusyInfo struct {
	Calendar string
	Busy     []TimeRange
	Errors   []string
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// AvailableSlot represents an available time slot for scheduling
type AvailableSlot struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// toEventSummary converts a Google Calendar event to an EventSummary
func toEventSummary(event *calendar.Event) EventSummary {
	summary := EventSummary{
		ID:          event.Id,
		Summary:     event.Summary,
		Description: event.Description,
		Location:    event.Location,
		Status:      event.Status,
		EventType:   event.EventType,
	}

	// Parse start time
	if event.Start != nil {
		if event.Start.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.Start.DateTime); err == nil {
				summary.Start = t
			}
		} else if event.Start.Date != "" {
			if t, err := time.Parse("2006-01-02", event.Start.Date); err == nil {
				summary.Start = t
			}
		}
	}

	// Parse end time
	if event.End != nil {
		if event.End.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.End.DateTime); err == nil {
				summary.End = t
			}
		} else if event.End.Date != "" {
			if t, err := time.Parse("2006-01-02", event.End.Date); err == nil {
				summary.End = t
			}
		}
	}

	// Creator and organizer
	if event.Creator != nil {
		summary.Creator = event.Creator.Email
	}
	if event.Organizer != nil {
		summary.Organizer = event.Organizer.Email
	}

	// Attendees
	for _, att := range event.Attendees {
		summary.Attendees = append(summary.Attendees, AttendeeInfo{
			Email:          att.Email,
			DisplayName:    att.DisplayName,
			ResponseStatus: att.ResponseStatus,
			Optional:       att.Optional,
			Organizer:      att.Organizer,
		})
	}

	// Google Meet link
	if event.ConferenceData != nil && len(event.ConferenceData.EntryPoints) > 0 {
		for _, ep := range event.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				summary.MeetLink = ep.Uri
				break
			}
		}
	}

	return summary
}

// toCalendarInfo converts a Google Calendar list entry to CalendarInfo
func toCalendarInfo(entry *calendar.CalendarListEntry) CalendarInfo {
	return CalendarInfo{
		ID:          entry.Id,
		Summary:     entry.Summary,
		Description: entry.Description,
		TimeZone:    entry.TimeZone,
		Primary:     entry.Primary,
		AccessRole:  entry.AccessRole,
	}
}
