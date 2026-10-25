package docparser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaddleOCRVLCloudTimeoutConfig(t *testing.T) {
	for _, tt := range []struct {
		name, value string
		want        time.Duration
	}{
		{"empty", "", paddleOCRVLCloudTimeout},
		{"whitespace", "  ", paddleOCRVLCloudTimeout},
		{"minutes", "90m", 90 * time.Minute},
		{"trimmed", " 5400s ", 90 * time.Minute},
		{"invalid", "invalid", paddleOCRVLCloudTimeout},
		{"unitless", "5400", paddleOCRVLCloudTimeout},
		{"zero", "0s", paddleOCRVLCloudTimeout},
		{"negative", "-1s", paddleOCRVLCloudTimeout},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WEKNORA_PADDLEOCR_VL_CLOUD_TIMEOUT", tt.value)
			assert.Equal(t, tt.want, NewPaddleOCRVLCloudReader(nil).timeout)
		})
	}
}

// The configured timeout, not the old hardcoded 600s, must end a job that the
// cloud keeps reporting as running.
func TestPaddleOCRVLCloudPollStopsAtConfiguredTimeout(t *testing.T) {
	allowLoopbackSSRF(t)
	t.Setenv("WEKNORA_PADDLEOCR_VL_CLOUD_TIMEOUT", "200ms")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"state":"running"}}`)
	}))
	defer server.Close()

	reader := NewPaddleOCRVLCloudReader(nil)
	reader.baseURL = server.URL
	reader.pollInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Watchdog: without the configured budget this loop keeps polling for the
	// old hardcoded 600s, which must fail the test instead of hanging it.
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, pollErr := reader.pollJob(ctx, "job-1")
		done <- pollErr
	}()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("timed out after %s", 200*time.Millisecond))
		assert.Less(t, time.Since(start), 5*time.Second)
	case <-time.After(5 * time.Second):
		t.Fatalf("poll loop ignored WEKNORA_PADDLEOCR_VL_CLOUD_TIMEOUT")
	}
}
