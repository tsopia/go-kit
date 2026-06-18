package llm

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestIsTransientModelError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"ctx canceled", context.Canceled, false},
		{"ctx deadline", context.DeadlineExceeded, false},
		{"wrapped ctx canceled", fmt.Errorf("call failed: %w", context.Canceled), false},

		{"429 rate limit", errors.New("http 429 Too Many Requests"), true},
		{"429 lowercase", errors.New("status 429: rate limit exceeded"), true},
		{"rate limit phrase", errors.New("rate limit exceeded"), true},
		{"too many requests", errors.New("too many requests"), true},
		{"503", errors.New("service unavailable: 503"), true},
		{"502", errors.New("bad gateway 502"), true},

		{"400 bad request", errors.New("400 bad request: invalid param"), false},
		{"401 unauthorized", errors.New("401 unauthorized"), false},
		{"403 forbidden", errors.New("403 forbidden"), false},
		{"404 not found", errors.New("404 not found: model does not exist"), false},
		{"generic error", errors.New("something went wrong"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTransientModelError(tc.err); got != tc.want {
				t.Errorf("isTransientModelError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
