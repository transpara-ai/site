package graph

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/transpara-ai/site/auth"
)

func validFactoryV1TestOrder(id, status string) FactoryV1Order {
	return FactoryV1Order{
		OrderID:        id,
		Version:        "1.0.0",
		Title:          "Order " + id,
		Channel:        "human_idea",
		SourceRef:      json.RawMessage(`{"kind":"human_idea","identity":"eventgraph:event_` + id + `","sha256":"` + strings.Repeat("d", 64) + `"}`),
		DocumentSHA256: strings.Repeat("a", 64),
		Status:         status,
		TLCStage:       "cfada",
		TLCIndex:       4,
		ElapsedMS:      65_000,
		Peers:          []string{"reviewer", "guardian"},
		GateState:      "pending",
		Evidence:       []FactoryV1Evidence{{Kind: "accepted_order", Ref: "eg:" + id, SHA256: strings.Repeat("b", 64)}},
		NextAction:     "complete the current gate",
		Budget:         FactoryV1Budget{MaxAttempts: 12, ConsumedAttempts: 4, RemainingAttempts: 8, MaxTokens: 100_000, ConsumedTokens: 20_000, RemainingTokens: 80_000},
		Stages:         []FactoryV1Stage{{Stage: "ingest_work", Index: 0, State: "passed", AttemptID: "attempt_1", EventID: "event_stage_1", OccurredAt: "2026-08-04T22:00:00Z", Peers: []string{"intake"}, WorkArtifactID: "artifact_1", Evidence: []FactoryV1Evidence{{Kind: "accepted_order", Reference: "eg:" + id}}}},
	}
}

func writeHealthyFactoryV1Projection(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	now := time.Now().UTC()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(FactoryV1Projection{
		SchemaVersion: factoryV1SchemaVersion,
		GeneratedAt:   now.Format(time.RFC3339Nano),
		Service: FactoryV1Service{
			ServiceID: "factory-v1", InstanceID: "factory-v1-test",
			StartedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), Status: "healthy", Healthy: true,
		},
	}); err != nil {
		t.Errorf("encode healthy projection: %v", err)
	}
}

func factoryV1TestApprovalReceipt(order FactoryV1Order, basis string) json.RawMessage {
	return json.RawMessage(`{"basis":"` + basis + `","actor_id":"eventgraph_human_1","credential_key_id":"key_1","source_sha256":"` + strings.Repeat("d", 64) + `","factory_order_blob_sha":"` + order.DocumentSHA256 + `","order_id":"` + order.OrderID + `","order_version":"` + order.Version + `","document_sha256":"` + order.DocumentSHA256 + `","approval_sentence":"Human approved this exact bounded order.","approval_source_event_id":"event_approval_1","issued_at":"2026-08-04T21:59:00Z"}`)
}

func TestFactoryV1MissionControlStates(t *testing.T) {
	now := time.Date(2026, 8, 4, 22, 0, 5, 0, time.UTC)
	progressing := validFactoryV1TestOrder("fo_progressing", "progressing")
	progressing.ActivelyExecuting = true
	progressing.ActiveAttemptID = "attempt_live"
	progressing.Evidence = nil
	progressing.Stages = []FactoryV1Stage{{Stage: "cfada", Index: 4, State: "running", AttemptID: "attempt_live", EventID: "event_running", OccurredAt: now.Add(-time.Second).Format(time.RFC3339), Peers: []string{"reviewer", "guardian"}}}
	blocked := validFactoryV1TestOrder("fo_blocked", "blocked")
	blocked.Blocker = "CFADA evidence not yet accepted"
	blocked.NextAction = "repair the exact design finding"
	human := validFactoryV1TestOrder("fo_human", "human_required")
	human.Blocker = "bounded operator decision required"
	human.NextAction = "resolve intervention int_1"
	ready := validFactoryV1TestOrder("fo_human_review", "human_review")
	ready.TLCStage = "human_review"
	ready.TLCIndex = 11
	ready.HumanApprovalBasis = "standing_scoped"
	ready.HumanApprovalReceipt = factoryV1TestApprovalReceipt(ready, "standing_scoped")
	ready.PR = FactoryV1PR{Repository: "transpara-ai/site", Number: 314, URL: "https://github.com/transpara-ai/site/pull/314", HeadSHA: strings.Repeat("c", 40), ReviewedHeadSHA: strings.Repeat("c", 40), Open: true, ChecksPassing: true}
	ready.NextAction = "Human reviews the exact ready PR head"

	projection := &FactoryV1Projection{
		SchemaVersion: factoryV1SchemaVersion,
		GeneratedAt:   now.Add(-5 * time.Second).Format(time.RFC3339),
		Service:       FactoryV1Service{ServiceID: "factory-v1", InstanceID: "factory-v1-demo", StartedAt: now.Add(-time.Hour).Format(time.RFC3339), RecoveryGeneration: 2, Healthy: true},
		Orders:        []FactoryV1Order{progressing, blocked, human, ready},
		Interventions: []FactoryV1Intervention{{InterventionID: "int_1", OrderID: "fo_human", Kind: "bounded_demo", Prompt: "Confirm the bounded correction", Status: "open", RequestedAt: now.Format(time.RFC3339), EventID: "event_int_1"}},
	}

	view := buildFactoryV1MissionControl(projection, nil, now)
	if view.Freshness != FreshnessCurrent || !view.Writable {
		t.Fatalf("freshness/writable = %q/%v, want current/true; notices=%v", view.Freshness, view.Writable, view.Notices)
	}
	if len(view.Orders) != 4 {
		t.Fatalf("orders = %d, want 4", len(view.Orders))
	}

	var out strings.Builder
	if err := factoryV1MissionControlSurface(view).Render(context.Background(), &out); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := out.String()
	for _, want := range []string{
		`data-console-surface="factory-v1"`,
		`data-order-id="fo_progressing"`, `data-order-status="progressing"`, "· active",
		`data-order-id="fo_blocked"`, `data-order-status="blocked"`, "CFADA evidence not yet accepted",
		`data-order-id="fo_human"`, `data-order-status="human_required"`, "bounded operator decision required",
		`data-order-id="fo_human_review"`, `data-order-status="human_review"`, "transpara-ai/site#314", "open exact-head ready", "passing",
		"reviewer, guardian", "1m5s", "4 / 12 attempts · 8 left", "20000 / 100000 tokens",
		`data-intervention-id="int_1"`, "Confirm the bounded correction", "Resolve and resume",
		`data-factory-v1-form="idea"`, `data-factory-v1-form="completed-order"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("render missing %q", want)
		}
	}

	brokenProjection := *projection
	brokenOrder := validFactoryV1TestOrder("fo_missing", "progressing")
	brokenOrder.DocumentSHA256 = ""
	brokenOrder.Evidence = nil
	brokenProjection.Orders = []FactoryV1Order{brokenOrder}
	broken := buildFactoryV1MissionControl(&brokenProjection, nil, now)
	if got := broken.Orders[0].EffectiveStatus; got != "blocked" {
		t.Fatalf("missing-evidence effective status = %q, want blocked", got)
	}
	if len(broken.Orders[0].Missing) == 0 {
		t.Fatal("missing-evidence order must list missing canonical fields")
	}

	unavailable := buildFactoryV1MissionControl(nil, io.EOF, now)
	out.Reset()
	if err := factoryV1MissionControlSurface(unavailable).Render(context.Background(), &out); err != nil {
		t.Fatalf("render unavailable: %v", err)
	}
	if !strings.Contains(out.String(), `data-state="unavailable"`) {
		t.Error("unavailable projection must render explicit unavailable state")
	}
	if strings.Contains(out.String(), `<form`) {
		t.Error("unavailable projection must not render mutation forms")
	}
}

func TestFactoryV1DecodesHiveProjectionContract(t *testing.T) {
	const fixture = `{
  "schema_version":"factory-v1",
  "generated_at":"2026-08-04T22:00:00Z",
  "service":{"service_id":"hive-factory-v1","instance_id":"demo-1","recovery_generation":3,"started_at":"2026-08-04T21:00:00Z","healthy":true,"detail":"ready"},
  "orders":[{
    "order_id":"FO-DEMO","version":"1.0.0","title":"Demo","channel":"human_idea",
    "source_ref":{"kind":"human_idea","identity":"human-idea:demo","sha256":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
    "document_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "status":"accepted","tlc_stage":"design","tlc_index":2,"elapsed_ms":1000,
    "actively_executing":false,"peers":["planner","reviewer"],"gate_state":"unavailable",
    "evidence":[{"kind":"design","reference":"docs/design.md","event_id":"phantom-evidence-event","design_blob_sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}],
    "next_action":"start design",
    "budget":{"max_attempts":24,"consumed_attempts":2,"remaining_attempts":22,"max_tokens":100000,"consumed_tokens":5000,"remaining_tokens":95000,"max_cost_micros":1000000,"consumed_cost_micros":100000,"remaining_cost_micros":900000,"exhausted":false},
    "stages":[{"stage":"craft_factory_order","index":1,"state":"passed","attempt_id":"attempt-2","ordinal":1,"event_id":"event-2","occurred_at":"2026-08-04T21:59:59Z","peers":["planner"],"evidence":[{"kind":"canonical_order","reference":"work:FO-DEMO"}],"work_artifact_id":"artifact-2","recovered":false}]
  }],
  "ideas":[{"idea_id":"idea-1","title":"Demo","target_repository":"transpara-ai/site","status":"refining","current_revision":1,"revisions":[{"revision":1,"note":"initial","candidate":{"doc_id":"FO-DEMO"},"candidate_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","validation_errors":[],"event_id":"event-idea","recorded_at":"2026-08-04T21:58:00Z"}]}],
  "interventions":[]
}`
	var projection FactoryV1Projection
	if err := json.Unmarshal([]byte(fixture), &projection); err != nil {
		t.Fatalf("decode Hive projection fixture: %v", err)
	}
	view := buildFactoryV1MissionControl(&projection, nil, time.Date(2026, 8, 4, 22, 0, 5, 0, time.UTC))
	if view.Freshness != FreshnessCurrent || !view.Writable {
		t.Fatalf("view = freshness %q writable %v notices %v", view.Freshness, view.Writable, view.Notices)
	}
	if got := view.Orders[0].EffectiveStatus; got != "accepted" {
		t.Fatalf("accepted Hive order maps to %q, want accepted", got)
	}
	if len(view.Orders[0].Missing) != 0 {
		t.Fatalf("valid Hive order reported missing fields: %v", view.Orders[0].Missing)
	}
	if got := factoryV1EvidenceReference(projection.Orders[0].Evidence[0]); got != "docs/design.md" {
		t.Fatalf("evidence reference = %q", got)
	}
	var rendered strings.Builder
	if err := factoryV1EvidenceItem(projection.Orders[0].Evidence[0]).Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered.String(), "phantom-evidence-event") || strings.Contains(rendered.String(), "· event") {
		t.Fatalf("evidence row retained phantom event id: %s", rendered.String())
	}
	if got := factoryV1IdeaRevisionInstruction(projection.Ideas[0].Revisions[0]); got != "initial" {
		t.Fatalf("idea note = %q", got)
	}
}

func TestFactoryV1EvidenceDueAtCanonicalStageBoundaries(t *testing.T) {
	direct := validFactoryV1TestOrder("FO-DIRECT", "accepted")
	direct.Channel = "completed_factory_order"
	direct.TLCStage = "write_code"
	direct.TLCIndex = 6
	direct.HumanApprovalBasis = ""
	direct.HumanApprovalReceipt = nil
	approval := factoryV1TestApprovalReceipt(direct, "standing_scoped")
	direct.Evidence = []FactoryV1Evidence{{Kind: "human_approval", Reference: "eventgraph:event_approval_1", Approval: approval}}
	direct.Stages = append(direct.Stages, FactoryV1Stage{
		Stage: "human_design_review", Index: 5, State: "passed", AttemptID: "attempt_human", EventID: "event_human", WorkArtifactID: "artifact_human", Peers: []string{"human", "guardian"},
		Evidence: []FactoryV1Evidence{{Kind: "human_approval", Reference: "eventgraph:event_approval_1", Approval: approval}},
	})
	if missing := factoryV1OrderMissingEvidence(direct); len(missing) != 0 {
		t.Fatalf("completed-order standing approval stage reported missing: %v", missing)
	}

	draftPending := direct
	draftPending.TLCStage = "create_draft_pr"
	draftPending.TLCIndex = 7
	draftPending.PR = FactoryV1PR{}
	if missing := factoryV1OrderMissingEvidence(draftPending); len(missing) != 0 {
		t.Fatalf("create_draft_pr requires evidence before it is due: %v", missing)
	}

	iar := direct
	iar.TLCStage = "iar"
	iar.TLCIndex = 8
	iar.PR = FactoryV1PR{Repository: "transpara-ai/work", Number: 97, HeadSHA: strings.Repeat("b", 40), Open: true, Draft: true}
	if missing := factoryV1OrderMissingEvidence(iar); len(missing) != 0 {
		t.Fatalf("IAR draft PR incorrectly requires later exact-head readiness: %v", missing)
	}

	markReady := iar
	markReady.TLCStage = "mark_pr_ready"
	markReady.TLCIndex = 10
	if missing := factoryV1OrderMissingEvidence(markReady); len(missing) != 0 {
		t.Fatalf("mark_pr_ready incorrectly requires its own terminal evidence: %v", missing)
	}

	humanReview := markReady
	humanReview.TLCStage = "human_review"
	humanReview.TLCIndex = 11
	humanReview.PR.Draft = false
	humanReview.PR.ChecksPassing = true
	humanReview.PR.ReviewedHeadSHA = humanReview.PR.HeadSHA
	if missing := factoryV1OrderMissingEvidence(humanReview); len(missing) != 0 {
		t.Fatalf("exact-head ready Human Review evidence reported missing: %v", missing)
	}
}

func TestFactoryV1FreshAcceptanceDoesNotInventFailure(t *testing.T) {
	order := validFactoryV1TestOrder("FO-FRESH", "accepted")
	order.TLCStage = "ingest_work"
	order.TLCIndex = 0
	order.Evidence = nil
	order.Stages = nil
	if missing := factoryV1OrderMissingEvidence(order); len(missing) != 0 {
		t.Fatalf("fresh accepted order reported missing future evidence: %v", missing)
	}
}

func TestFactoryV1InterventionPOST(t *testing.T) {
	type capturedRequest struct {
		path          string
		authorization string
		contentType   string
		body          map[string]any
	}
	captured := make(chan capturedRequest, 1)
	hive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeHealthyFactoryV1Projection(t, w)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		captured <- capturedRequest{path: r.URL.Path, authorization: r.Header.Get("Authorization"), contentType: r.Header.Get("Content-Type"), body: body}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"event_ids":["event_resolved"]}`))
	}))
	defer hive.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hive.URL)
	t.Setenv("HIVE_OPS_API_KEY", "operator-secret")
	t.Setenv("HIVE_FACTORY_V1_ACTOR_ID", "eventgraph_human_1")

	operator := &auth.User{ID: "human_operator_1", Name: "Operator", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.ContextWithUser(r.Context(), operator)))
		})
	}
	h := NewHandlers(nil, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	form := url.Values{"resolution": {"Use the verified bounded input and resume this order."}}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/interventions/int_1/resolve", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", w.Code, w.Body.String())
	}
	if location := w.Header().Get("Location"); location != "/console/factory-v1" {
		t.Fatalf("Location = %q, want /console/factory-v1", location)
	}

	got := <-captured
	if got.path != "/api/hive/factory/v1/interventions/int_1/resolve" {
		t.Errorf("path = %q", got.path)
	}
	if got.authorization != "Bearer operator-secret" || got.contentType != "application/json" {
		t.Errorf("headers = Authorization:%q Content-Type:%q", got.authorization, got.contentType)
	}
	if got.body["actor_id"] != "eventgraph_human_1" || got.body["resolution"] != "Use the verified bounded input and resume this order." {
		t.Errorf("body = %#v", got.body)
	}
	if got.body["operator_principal_id"] != "human_operator_1" {
		t.Errorf("operator principal = %#v, want authenticated human_operator_1", got.body["operator_principal_id"])
	}
}

func TestFactoryV1IdeaAndCompletedOrderPOST(t *testing.T) {
	type captured struct {
		path string
		body map[string]any
	}
	requests := make(chan captured, 4)
	hive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeHealthyFactoryV1Projection(t, w)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		requests <- captured{path: r.URL.Path, body: body}
		w.WriteHeader(http.StatusCreated)
	}))
	defer hive.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hive.URL)
	t.Setenv("HIVE_OPS_API_KEY", "operator-secret")

	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	ideaForm := url.Values{"title": {"Demo idea"}, "idea": {"Make progress visible"}, "target_repository": {"transpara-ai/site"}}
	ideaReq := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/ideas", strings.NewReader(ideaForm.Encode()))
	ideaReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ideaW := httptest.NewRecorder()
	mux.ServeHTTP(ideaW, ideaReq)
	if ideaW.Code != http.StatusSeeOther {
		t.Fatalf("idea status = %d; body=%s", ideaW.Code, ideaW.Body.String())
	}

	refineForm := url.Values{"instruction": {"Make the acceptance evidence exact-head."}}
	refineReq := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/ideas/idea_1/refine", strings.NewReader(refineForm.Encode()))
	refineReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	refineW := httptest.NewRecorder()
	mux.ServeHTTP(refineW, refineReq)
	if refineW.Code != http.StatusSeeOther {
		t.Fatalf("refine status = %d; body=%s", refineW.Code, refineW.Body.String())
	}

	candidateSHA256 := strings.Repeat("c", 64)
	submitForm := url.Values{"revision": {"2"}, "candidate_sha256": {candidateSHA256}}
	submitReq := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/ideas/idea_1/submit", strings.NewReader(submitForm.Encode()))
	submitReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	submitW := httptest.NewRecorder()
	mux.ServeHTTP(submitW, submitReq)
	if submitW.Code != http.StatusSeeOther {
		t.Fatalf("submit status = %d; body=%s", submitW.Code, submitW.Body.String())
	}

	orderForm := url.Values{"factory_order": {`{"doc_id":"FO-DEMO","version":"1.0.0","status":"approved","title":"Demo"}`}}
	orderReq := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/orders", strings.NewReader(orderForm.Encode()))
	orderReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	orderW := httptest.NewRecorder()
	mux.ServeHTTP(orderW, orderReq)
	if orderW.Code != http.StatusSeeOther {
		t.Fatalf("order status = %d; body=%s", orderW.Code, orderW.Body.String())
	}

	byPath := make(map[string]map[string]any, 4)
	for range 4 {
		item := <-requests
		byPath[item.path] = item.body
	}
	if got := byPath["/api/hive/factory/v1/ideas"]; got["title"] != "Demo idea" || got["idea"] != "Make progress visible" || got["target_repository"] != "transpara-ai/site" {
		t.Errorf("idea payload = %#v", got)
	}
	gotOrder, ok := byPath["/api/hive/factory/v1/orders"]["factory_order"].(map[string]any)
	if !ok || gotOrder["doc_id"] != "FO-DEMO" {
		t.Errorf("order payload = %#v", byPath["/api/hive/factory/v1/orders"])
	}
	if got := byPath["/api/hive/factory/v1/ideas/idea_1/refine"]; got["instruction"] != "Make the acceptance evidence exact-head." {
		t.Errorf("refine payload = %#v", got)
	}
	if got := byPath["/api/hive/factory/v1/ideas/idea_1/submit"]; got["approved"] != true || got["revision"] != float64(2) || got["candidate_sha256"] != candidateSHA256 {
		t.Errorf("submit payload = %#v", got)
	}
}

func TestFactoryV1MutationRequiresFreshWritableProjection(t *testing.T) {
	now := time.Now().UTC()
	healthy := FactoryV1Projection{
		SchemaVersion: factoryV1SchemaVersion,
		GeneratedAt:   now.Format(time.RFC3339Nano),
		Service: FactoryV1Service{
			ServiceID: "factory-v1", InstanceID: "factory-v1-test",
			StartedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), Status: "healthy", Healthy: true,
		},
	}
	tests := []struct {
		name       string
		projection FactoryV1Projection
	}{
		{name: "unsupported schema", projection: func() FactoryV1Projection { p := healthy; p.SchemaVersion = "future"; return p }()},
		{name: "stale projection", projection: func() FactoryV1Projection {
			p := healthy
			p.GeneratedAt = now.Add(-10 * time.Minute).Format(time.RFC3339Nano)
			return p
		}()},
		{name: "unhealthy service", projection: func() FactoryV1Projection {
			p := healthy
			p.Service.Status = "stopped"
			p.Service.Healthy = false
			return p
		}()},
		{name: "contradictory healthy status", projection: func() FactoryV1Projection {
			p := healthy
			p.Service.Healthy = false
			return p
		}()},
		{name: "missing service identity", projection: func() FactoryV1Projection {
			p := healthy
			p.Service.ServiceID = ""
			p.Service.InstanceID = ""
			return p
		}()},
		{name: "missing service start", projection: func() FactoryV1Projection { p := healthy; p.Service.StartedAt = ""; return p }()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			posts := 0
			hive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posts++
					w.WriteHeader(http.StatusCreated)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(test.projection); err != nil {
					t.Errorf("encode projection: %v", err)
				}
			}))
			defer hive.Close()
			t.Setenv("HIVE_OPS_API_BASE_URL", hive.URL)
			t.Setenv("HIVE_OPS_API_KEY", "operator-secret")

			h := NewHandlers(nil, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/ideas", nil)
			w := httptest.NewRecorder()
			h.factoryV1Mutation(w, req, "/api/hive/factory/v1/ideas", map[string]any{"title": "must not post"})
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503; body=%s", w.Code, w.Body.String())
			}
			if posts != 0 {
				t.Fatalf("POST count = %d, want 0", posts)
			}
		})
	}

	t.Run("projection transport failure", func(t *testing.T) {
		hive := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		base := hive.URL
		hive.Close()
		t.Setenv("HIVE_OPS_API_BASE_URL", base)
		t.Setenv("HIVE_OPS_API_KEY", "operator-secret")
		h := NewHandlers(nil, nil, nil)
		req := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/ideas", nil)
		w := httptest.NewRecorder()
		h.factoryV1Mutation(w, req, "/api/hive/factory/v1/ideas", map[string]any{"title": "must not post"})
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503; body=%s", w.Code, w.Body.String())
		}
	})
}

func TestFactoryV1AnonymousPrincipalCannotBorrowConfiguredActor(t *testing.T) {
	t.Setenv("HIVE_FACTORY_V1_ACTOR_ID", "eventgraph_human_1")
	h := NewHandlers(nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/interventions/int_1/resolve", nil)
	actorID, principalID, ok := h.factoryV1ActorIdentity(req)
	if ok || actorID != "" || principalID != "" {
		t.Fatalf("anonymous identity = (%q, %q, %v), want rejected", actorID, principalID, ok)
	}
}

func TestFactoryV1IdeaSubmitRequiresExactCandidate(t *testing.T) {
	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	for name, form := range map[string]url.Values{
		"missing revision": {"candidate_sha256": {strings.Repeat("a", 64)}},
		"invalid digest":   {"revision": {"3"}, "candidate_sha256": {"not-a-digest"}},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://site.test/console/factory-v1/ideas/idea_1/submit", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestFactoryV1NotRegisteredInNoDBConsole(t *testing.T) {
	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterReadOnlyConsole(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/factory-v1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
