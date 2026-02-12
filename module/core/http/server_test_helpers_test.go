package http

import (
	stdhttp "net/http"
	"testing"
	"time"
)

// waitForHTTPReady retries a GET request until endpoint becomes available.
func waitForHTTPReady(t *testing.T, url string, attempts int, interval time.Duration) {
	t.Helper()

	client := &stdhttp.Client{Timeout: 250 * time.Millisecond}
	for index := 0; index < attempts; index++ {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == stdhttp.StatusOK {
				return
			}
		}

		time.Sleep(interval)
	}

	t.Fatalf("endpoint %s did not become ready after %d attempts", url, attempts)
}

// waitForCondition retries condition checks until they pass or attempts are exhausted.
func waitForCondition(t *testing.T, attempts int, interval time.Duration, condition func() bool) {
	t.Helper()

	for index := 0; index < attempts; index++ {
		if condition() {
			return
		}

		time.Sleep(interval)
	}

	t.Fatalf("condition did not become true after %d attempts", attempts)
}
