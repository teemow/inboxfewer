package instrumentation

import "testing"

func TestExtractUserDomain(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"jane@example.com", "example.com"},
		{"user@gmail.com", "gmail.com"},
		{"admin@company.org", "company.org"},
		{"test@subdomain.example.com", "subdomain.example.com"},
		{"invalid", "unknown"},
		{"", "unknown"},
		{"@", "unknown"},
		{"user@", "unknown"},
		{"@domain.com", "domain.com"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := ExtractUserDomain(tt.email)
			if result != tt.expected {
				t.Errorf("ExtractUserDomain(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestOperationConstants(t *testing.T) {
	operations := map[string]string{
		OperationList:   "list",
		OperationGet:    "get",
		OperationCreate: "create",
		OperationUpdate: "update",
		OperationDelete: "delete",
		OperationSend:   "send",
		OperationSearch: "search",
	}

	for constant, expected := range operations {
		if constant != expected {
			t.Errorf("Operation constant = %q, want %q", constant, expected)
		}
	}
}
