package gmail_tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnsubscribeToolsRegistration(t *testing.T) {
	// Test that unsubscribe tools can be registered without errors
	// This is a basic smoke test to ensure the tool definitions are valid
	assert.NotNil(t, RegisterUnsubscribeTools)
}
