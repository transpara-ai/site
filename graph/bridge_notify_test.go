// site/graph/bridge_notify_test.go
package graph

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestTeamsNotifier(t *testing.T) {
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := &TeamsNotifier{WebhookURL: srv.URL}
	err := notifier.Send("SDR needs approval: outbound to Sarah Chen", "https://site.example.com/bridge/actions/123")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !called.Load() {
		t.Error("webhook not called")
	}
}

func TestEmailNotifierFormat(t *testing.T) {
	notifier := &EmailNotifier{
		From:    "bridge@transpara-ai.dev",
		To:      "matt@transpara.com",
		Subject: "[Transpara-AI] SDR needs approval",
	}

	body := notifier.FormatBody("SDR needs approval: outbound to Sarah Chen", "https://site.example.com/bridge/actions/123")
	if body == "" {
		t.Fatal("body should not be empty")
	}
}

func TestNotifyDispatcherDedup(t *testing.T) {
	dispatcher := NewNotifyDispatcher()

	// First call should not be deduped
	if dispatcher.IsDuplicate("action-1") {
		t.Error("first call should not be duplicate")
	}

	// Second call within window should be deduped
	if !dispatcher.IsDuplicate("action-1") {
		t.Error("second call should be duplicate")
	}

	// Different action should not be deduped
	if dispatcher.IsDuplicate("action-2") {
		t.Error("different action should not be duplicate")
	}
}
