package graph

import (
	"fmt"
	"time"
)

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func prefEnabled(prefs []NotifyPreference, channel string) bool {
	for _, p := range prefs {
		if p.Channel == channel {
			return p.Enabled
		}
	}
	return false
}
