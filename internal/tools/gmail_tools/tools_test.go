package gmail_tools

import (
	"testing"
)

// TestToolsPackage ensures the package compiles and basic functionality works
func TestToolsPackage(t *testing.T) {
	// Test that getAccountFromArgs works correctly
	args := map[string]interface{}{
		"account": "test-account",
	}
	account := getAccountFromArgs(args)
	if account != "test-account" {
		t.Errorf("getAccountFromArgs() = %v, want test-account", account)
	}

	// Test default account
	emptyArgs := map[string]interface{}{}
	defaultAccount := getAccountFromArgs(emptyArgs)
	if defaultAccount != "default" {
		t.Errorf("getAccountFromArgs() = %v, want default", defaultAccount)
	}
}
