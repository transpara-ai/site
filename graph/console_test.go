package graph

import (
	"errors"
	"testing"
	"time"
)

func TestDeriveFreshness(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	rfc := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339) }

	tests := []struct {
		name             string
		generatedAt      string
		fetchErr         error
		hasPartialErrors bool
		want             ConsoleFreshness
	}{
		{"fetch error is unavailable", rfc(-1 * time.Second), errors.New("down"), false, FreshnessUnavailable},
		{"empty timestamp is unavailable", "", nil, false, FreshnessUnavailable},
		{"unparseable timestamp is unavailable", "not-a-time", nil, false, FreshnessUnavailable},
		{"older than window is stale", rfc(-90 * time.Second), nil, false, FreshnessStale},
		{"fresh with partial errors is partial", rfc(-2 * time.Second), nil, true, FreshnessPartial},
		{"fresh and clean is current", rfc(-2 * time.Second), nil, false, FreshnessCurrent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveFreshness(tt.generatedAt, tt.fetchErr, tt.hasPartialErrors, now, consoleStaleWindow)
			if got != tt.want {
				t.Fatalf("deriveFreshness = %q, want %q", got, tt.want)
			}
		})
	}
}
