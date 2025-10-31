package gmail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseListUnsubscribe(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected []UnsubscribeMethod
	}{
		{
			name:   "single mailto",
			header: "<mailto:unsubscribe@example.com>",
			expected: []UnsubscribeMethod{
				{Type: "mailto", URL: "mailto:unsubscribe@example.com"},
			},
		},
		{
			name:   "single http",
			header: "<https://example.com/unsubscribe>",
			expected: []UnsubscribeMethod{
				{Type: "http", URL: "https://example.com/unsubscribe"},
			},
		},
		{
			name:   "mailto with subject",
			header: "<mailto:unsubscribe@example.com?subject=unsubscribe>",
			expected: []UnsubscribeMethod{
				{Type: "mailto", URL: "mailto:unsubscribe@example.com?subject=unsubscribe"},
			},
		},
		{
			name:   "multiple methods",
			header: "<mailto:unsubscribe@example.com>, <https://example.com/unsubscribe>",
			expected: []UnsubscribeMethod{
				{Type: "mailto", URL: "mailto:unsubscribe@example.com"},
				{Type: "http", URL: "https://example.com/unsubscribe"},
			},
		},
		{
			name:   "http only",
			header: "<http://example.com/unsubscribe>",
			expected: []UnsubscribeMethod{
				{Type: "http", URL: "http://example.com/unsubscribe"},
			},
		},
		{
			name:     "empty header",
			header:   "",
			expected: nil,
		},
		{
			name:   "multiple http methods",
			header: "<https://example.com/unsubscribe?id=123>, <https://example.com/unsubscribe-alt>",
			expected: []UnsubscribeMethod{
				{Type: "http", URL: "https://example.com/unsubscribe?id=123"},
				{Type: "http", URL: "https://example.com/unsubscribe-alt"},
			},
		},
		{
			name:   "with extra whitespace",
			header: " < mailto:unsubscribe@example.com > , < https://example.com/unsub > ",
			expected: []UnsubscribeMethod{
				{Type: "mailto", URL: "mailto:unsubscribe@example.com"},
				{Type: "http", URL: "https://example.com/unsub"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseListUnsubscribe(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnsubscribeInfo(t *testing.T) {
	// Test that UnsubscribeInfo is properly structured
	info := &UnsubscribeInfo{
		MessageID:      "msg123",
		HasUnsubscribe: true,
		Methods: []UnsubscribeMethod{
			{Type: "mailto", URL: "mailto:unsub@example.com"},
			{Type: "http", URL: "https://example.com/unsub"},
		},
	}

	assert.Equal(t, "msg123", info.MessageID)
	assert.True(t, info.HasUnsubscribe)
	assert.Len(t, info.Methods, 2)
	assert.Equal(t, "mailto", info.Methods[0].Type)
	assert.Equal(t, "http", info.Methods[1].Type)
}
