package utils

import "testing"

func TestStreamingKey(t *testing.T) {
	if StreamingKey != "stream" {
		t.Errorf("StreamingKey = %q, want %q", StreamingKey, "stream")
	}
}
