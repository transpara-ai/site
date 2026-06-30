package graph

import "time"

type ConsoleFreshness string

const (
	FreshnessCurrent     ConsoleFreshness = "current"
	FreshnessStale       ConsoleFreshness = "stale"
	FreshnessPartial     ConsoleFreshness = "partial"
	FreshnessUnavailable ConsoleFreshness = "unavailable"
)

const consoleStaleWindow = 30 * time.Second

// deriveFreshness maps upstream signals onto an explicit freshness state.
// It fails closed: a fetch error, an empty or unparseable timestamp, or any
// other ambiguity resolves to FreshnessUnavailable. Only a parseable,
// within-window, error-free projection earns FreshnessCurrent.
func deriveFreshness(generatedAt string, fetchErr error, hasPartialErrors bool, now time.Time, staleWindow time.Duration) ConsoleFreshness {
	if fetchErr != nil {
		return FreshnessUnavailable
	}
	ts, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return FreshnessUnavailable
	}
	if now.Sub(ts) > staleWindow {
		return FreshnessStale
	}
	if hasPartialErrors {
		return FreshnessPartial
	}
	return FreshnessCurrent
}
