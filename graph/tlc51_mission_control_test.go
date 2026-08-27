package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTLC51MissionControlFetchRequiresBearerAndRejectsAuthority(t *testing.T) {
	now := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	projection := missionTLC51TestProjection(now)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != tlc51MissionControlPath || r.Header.Get("Authorization") != "Bearer exact-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(projection)
	}))
	defer server.Close()
	a := &missionControlAcquirer{client: server.Client(), now: func() time.Time { return now }}

	t.Setenv("HIVE_OPS_API_KEY", "")
	if _, err := a.fetchTLC51(context.Background(), server.URL+tlc51MissionControlPath, now); err == nil || !strings.Contains(err.Error(), "API_KEY") {
		t.Fatalf("missing-key error = %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("request sent without bearer: %d", requests.Load())
	}

	t.Setenv("HIVE_OPS_API_KEY", "exact-token")
	got, err := a.fetchTLC51(context.Background(), server.URL+tlc51MissionControlPath, now)
	if err != nil || len(got.Orders) != 1 {
		t.Fatalf("exact projection = %+v error=%v", got, err)
	}

	projection.AuthorityGranted = true
	if _, err := a.fetchTLC51(context.Background(), server.URL+tlc51MissionControlPath, now); err == nil || !strings.Contains(err.Error(), "must not grant authority") {
		t.Fatalf("authority-bearing projection error = %v", err)
	}
}

func TestTLC51MissionControlRejectsUnknownFieldsAndDuplicateRows(t *testing.T) {
	now := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	t.Setenv("HIVE_OPS_API_KEY", "exact-token")
	unknown := `{"schema_version":"factory-tlc51-mission-control-envelope/v1","generated_at":"2026-08-27T19:00:00Z","orders":[],"errors":[],"authority_granted":false,"invented_policy":"pass"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(unknown)) }))
	a := &missionControlAcquirer{client: server.Client(), now: func() time.Time { return now }}
	if _, err := a.fetchTLC51(context.Background(), server.URL, now); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	server.Close()

	projection := missionTLC51TestProjection(now)
	projection.Orders = append(projection.Orders, projection.Orders[0])
	if err := validateTLC51MissionControl(projection, now); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate-row error = %v", err)
	}
}

func TestTLC51MissionControlRejectsBlankExternalState(t *testing.T) {
	now := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	projection := missionTLC51TestProjection(now)
	projection.Orders[0].Effects[0].ExternalState = ""
	if err := validateTLC51MissionControl(projection, now); err == nil || !strings.Contains(err.Error(), "invalid identity") {
		t.Fatalf("blank external state error = %v", err)
	}
}

func TestTLC51MissionControlRendersUnclassifiedWithoutLegacyDefault(t *testing.T) {
	now := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	projection := missionTLC51TestProjection(now)
	projection.Orders[0].InformationState = "UNCLASSIFIED"
	projection.Orders[0].Track = nil
	projection.Orders[0].RetainedFloor = nil
	projection.Orders[0].Blockers = []string{"information_state:UNCLASSIFIED"}
	view := MissionControlView{
		TLC51Projection: &projection, TLC51Acquisition: missionTestMark(now, "exact"),
		HiveAcquisition: missionTestUnavailable(now), WorkHealth: MissionObservedService{}, SiteHealth: MissionObservedService{},
		GeneratedAt: now, OverallStatus: "unavailable",
	}
	var rendered bytes.Buffer
	if err := missionControlFragment(view).Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	body := rendered.String()
	for _, want := range []string{"UNCLASSIFIED", "track unknown", "retained unknown", "information_state:UNCLASSIFIED", "factory-tlc51/v1"} {
		if !strings.Contains(body, want) {
			t.Errorf("render missing %q", want)
		}
	}
	if strings.Contains(body, "P-ENVELOPE") || strings.Contains(body, "TLC 4.5 packet") {
		t.Fatal("unclassified TLC 5.1 row inherited a legacy policy default")
	}
}
