// site/graph/bridge_notify.go
package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Notifier sends a notification to a specific channel.
type Notifier interface {
	Send(message, link string) error
}

// TeamsNotifier sends notifications via Teams incoming webhook.
type TeamsNotifier struct {
	WebhookURL string
}

func (n *TeamsNotifier) Send(message, link string) error {
	payload := map[string]interface{}{
		"@type":    "MessageCard",
		"summary":  message,
		"sections": []map[string]string{
			{"activityTitle": message, "activitySubtitle": link},
		},
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(n.WebhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("teams webhook: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("teams webhook returned %d", resp.StatusCode)
	}
	return nil
}

// EmailNotifier formats email notifications. Actual sending is deferred to SMTP integration.
type EmailNotifier struct {
	From    string
	To      string
	Subject string
}

func (n *EmailNotifier) FormatBody(message, link string) string {
	return fmt.Sprintf("%s\n\nReview and decide: %s\n\n-- Transpara-AI Agent Bridge", message, link)
}

func (n *EmailNotifier) Send(message, link string) error {
	// SMTP sending deferred to Phase 2b integration.
	// For now, log the notification.
	fmt.Printf("[EMAIL] To: %s | %s | %s\n", n.To, message, link)
	return nil
}

// NotifyDispatcher manages deduplication and channel routing.
type NotifyDispatcher struct {
	mu      sync.Mutex
	sent    map[string]time.Time
	window  time.Duration
}

// NewNotifyDispatcher creates a dispatcher with a 5-minute dedup window.
func NewNotifyDispatcher() *NotifyDispatcher {
	return &NotifyDispatcher{
		sent:   make(map[string]time.Time),
		window: 5 * time.Minute,
	}
}

// IsDuplicate returns true if this action was already notified within the dedup window.
func (d *NotifyDispatcher) IsDuplicate(actionID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if t, ok := d.sent[actionID]; ok && time.Since(t) < d.window {
		return true
	}
	d.sent[actionID] = time.Now()
	return false
}
