package batch

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParseStringOrArray(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		paramName string
		want      []string
		wantErr   bool
	}{
		{
			name:      "single string",
			input:     "test123",
			paramName: "testParam",
			want:      []string{"test123"},
			wantErr:   false,
		},
		{
			name:      "array of strings",
			input:     []interface{}{"id1", "id2", "id3"},
			paramName: "testParam",
			want:      []string{"id1", "id2", "id3"},
			wantErr:   false,
		},
		{
			name:      "nil input",
			input:     nil,
			paramName: "testParam",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "empty string",
			input:     "",
			paramName: "testParam",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "empty array",
			input:     []interface{}{},
			paramName: "testParam",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "array with non-string",
			input:     []interface{}{"id1", 123, "id3"},
			paramName: "testParam",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "array with empty string",
			input:     []interface{}{"id1", "", "id3"},
			paramName: "testParam",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "invalid type",
			input:     123,
			paramName: "testParam",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "JSON string array",
			input:     `["id1", "id2", "id3"]`,
			paramName: "testParam",
			want:      []string{"id1", "id2", "id3"},
			wantErr:   false,
		},
		{
			name:      "JSON string array with filenames",
			input:     `["document1.pdf", "document2.pdf", "document3.pdf"]`,
			paramName: "testParam",
			want:      []string{"document1.pdf", "document2.pdf", "document3.pdf"},
			wantErr:   false,
		},
		{
			name:      "JSON string single element array",
			input:     `["single.pdf"]`,
			paramName: "testParam",
			want:      []string{"single.pdf"},
			wantErr:   false,
		},
		{
			name:      "JSON string empty array",
			input:     `[]`,
			paramName: "testParam",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "invalid JSON string",
			input:     `[invalid json`,
			paramName: "testParam",
			want:      []string{`[invalid json`},
			wantErr:   false,
		},
		{
			name:      "string starting with bracket (not JSON)",
			input:     `[test] file.pdf`,
			paramName: "testParam",
			want:      []string{`[test] file.pdf`},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStringOrArray(tt.input, tt.paramName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStringOrArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !stringSliceEqual(got, tt.want) {
				t.Errorf("ParseStringOrArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	results := []Result{
		{ID: "id1", Status: "success", Result: "Operation successful"},
		{ID: "id2", Status: "success", Result: "Operation successful"},
		{ID: "id3", Status: "error", Error: "Something went wrong"},
	}

	output := FormatResults(results)

	var br BatchResult
	if err := json.Unmarshal([]byte(output), &br); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if br.Total != 3 {
		t.Errorf("Total = %d, want 3", br.Total)
	}
	if br.Successful != 2 {
		t.Errorf("Successful = %d, want 2", br.Successful)
	}
	if br.Failed != 1 {
		t.Errorf("Failed = %d, want 1", br.Failed)
	}
	if len(br.Results) != 3 {
		t.Errorf("len(Results) = %d, want 3", len(br.Results))
	}
}

func TestProcessBatch(t *testing.T) {
	ids := []string{"id1", "id2", "id3"}

	// Mock function that fails on id2
	fn := func(id string) (string, error) {
		if id == "id2" {
			return "", errors.New("failed to process id2")
		}
		return "processed " + id, nil
	}

	results := ProcessBatch(ids, fn)

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	// Check id1 - success
	if results[0].Status != "success" {
		t.Errorf("results[0].Status = %s, want success", results[0].Status)
	}
	if results[0].Result != "processed id1" {
		t.Errorf("results[0].Result = %s, want 'processed id1'", results[0].Result)
	}

	// Check id2 - error
	if results[1].Status != "error" {
		t.Errorf("results[1].Status = %s, want error", results[1].Status)
	}
	if results[1].Error != "failed to process id2" {
		t.Errorf("results[1].Error = %s, want 'failed to process id2'", results[1].Error)
	}

	// Check id3 - success
	if results[2].Status != "success" {
		t.Errorf("results[2].Status = %s, want success", results[2].Status)
	}
	if results[2].Result != "processed id3" {
		t.Errorf("results[2].Result = %s, want 'processed id3'", results[2].Result)
	}
}

func TestNewSuccessResult(t *testing.T) {
	result := NewSuccessResult("test-id", "test message")

	if result.ID != "test-id" {
		t.Errorf("ID = %s, want test-id", result.ID)
	}
	if result.Status != "success" {
		t.Errorf("Status = %s, want success", result.Status)
	}
	if result.Result != "test message" {
		t.Errorf("Result = %s, want 'test message'", result.Result)
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got %s", result.Error)
	}
}

func TestNewErrorResult(t *testing.T) {
	err := errors.New("test error")
	result := NewErrorResult("test-id", err)

	if result.ID != "test-id" {
		t.Errorf("ID = %s, want test-id", result.ID)
	}
	if result.Status != "error" {
		t.Errorf("Status = %s, want error", result.Status)
	}
	if result.Error != "test error" {
		t.Errorf("Error = %s, want 'test error'", result.Error)
	}
	if result.Result != "" {
		t.Errorf("Result should be empty, got %s", result.Result)
	}
}

// Helper function to compare string slices
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
