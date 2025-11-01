package batch

import (
	"encoding/json"
	"fmt"
)

// Result represents the result of a single operation in a batch
type Result struct {
	ID     string `json:"id"`
	Status string `json:"status"` // "success" or "error"
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// BatchResult represents the aggregated results of a batch operation
type BatchResult struct {
	Total      int      `json:"total"`
	Successful int      `json:"successful"`
	Failed     int      `json:"failed"`
	Results    []Result `json:"results"`
}

// ParseStringOrArray parses a parameter that can be either a single string or an array of strings
func ParseStringOrArray(param interface{}, paramName string) ([]string, error) {
	if param == nil {
		return nil, fmt.Errorf("%s is required", paramName)
	}

	var result []string

	switch v := param.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("%s cannot be empty", paramName)
		}
		result = []string{v}
	case []interface{}:
		if len(v) == 0 {
			return nil, fmt.Errorf("%s cannot be empty", paramName)
		}
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s[%d] must be a string", paramName, i)
			}
			if str == "" {
				return nil, fmt.Errorf("%s[%d] cannot be empty", paramName, i)
			}
			result = append(result, str)
		}
	default:
		return nil, fmt.Errorf("%s must be a string or array of strings", paramName)
	}

	return result, nil
}

// FormatResults creates a formatted JSON string from batch results
func FormatResults(results []Result) string {
	br := BatchResult{
		Total:   len(results),
		Results: results,
	}

	for _, r := range results {
		if r.Status == "success" {
			br.Successful++
		} else {
			br.Failed++
		}
	}

	jsonBytes, _ := json.MarshalIndent(br, "", "  ")
	return string(jsonBytes)
}

// ProcessBatch executes a function on each item and collects results
// fn should return (result string, error) for each item
func ProcessBatch(ids []string, fn func(id string) (string, error)) []Result {
	results := make([]Result, 0, len(ids))

	for _, id := range ids {
		result := Result{ID: id}
		res, err := fn(id)
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
		} else {
			result.Status = "success"
			result.Result = res
		}
		results = append(results, result)
	}

	return results
}

// NewSuccessResult creates a success result
func NewSuccessResult(id, message string) Result {
	return Result{
		ID:     id,
		Status: "success",
		Result: message,
	}
}

// NewErrorResult creates an error result
func NewErrorResult(id string, err error) Result {
	return Result{
		ID:     id,
		Status: "error",
		Error:  err.Error(),
	}
}
