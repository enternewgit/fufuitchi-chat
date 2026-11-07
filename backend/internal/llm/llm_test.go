package llm

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/genai"
)

func TestIsQuotaExceeded(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{genai.APIError{Code: 429, Status: "RESOURCE_EXHAUSTED"}, true},
		{&genai.APIError{Code: 429}, true},
		{fmt.Errorf("wrapped: %w", genai.APIError{Status: "RESOURCE_EXHAUSTED"}), true},
		{genai.APIError{Code: 404, Status: "NOT_FOUND"}, false},
		{errors.New("network error"), false},
	}
	for _, tt := range tests {
		if got := isQuotaExceeded(tt.err); got != tt.want {
			t.Errorf("%v: got %v, want %v", tt.err, got, tt.want)
		}
	}
}
