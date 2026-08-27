package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	tlc51MissionControlSchema = "factory-tlc51-mission-control-envelope/v1"
	tlc51MissionControlPath   = "/api/hive/factory-tlc51/mission-control"
)

type TLC51MissionControlEnvelope struct {
	SchemaVersion    string                     `json:"schema_version"`
	GeneratedAt      time.Time                  `json:"generated_at"`
	Orders           []TLC51MissionControlOrder `json:"orders"`
	Errors           []string                   `json:"errors"`
	AuthorityGranted bool                       `json:"authority_granted"`
}

type TLC51MissionControlOrder struct {
	SchemaVersion        string                   `json:"schema_version"`
	ProtocolVersion      string                   `json:"protocol_version"`
	FactoryOrderID       string                   `json:"factory_order_id"`
	OrderID              string                   `json:"order_id"`
	OrderVersion         string                   `json:"order_version"`
	ChangeSeriesID       string                   `json:"change_series_id"`
	ReleaseIdentity      json.RawMessage          `json:"release_identity"`
	AdapterIdentity      json.RawMessage          `json:"adapter_identity"`
	Repository           string                   `json:"repository"`
	InformationState     string                   `json:"information_state"`
	Track                *string                  `json:"track"`
	RetainedFloor        *string                  `json:"retained_floor"`
	Subject              json.RawMessage          `json:"subject"`
	SubjectDigest        string                   `json:"subject_digest"`
	PlanDigest           string                   `json:"plan_digest"`
	Obligations          []TLC51MissionObligation `json:"obligations"`
	Blockers             []string                 `json:"blockers"`
	HumanWaits           []TLC51MissionHumanWait  `json:"human_waits"`
	Decision             string                   `json:"decision"`
	ReceiptDigest        string                   `json:"receipt_digest,omitempty"`
	Effects              []TLC51MissionEffect     `json:"effects"`
	WorkReconciliation   string                   `json:"work_reconciliation"`
	EventGraphEventCount int                      `json:"eventgraph_event_count"`
	WorkArtifactCount    int                      `json:"work_artifact_count"`
	GeneratedAt          time.Time                `json:"generated_at"`
	AuthorityGranted     bool                     `json:"authority_granted"`
}

type TLC51MissionObligation struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Prerequisites     []string `json:"prerequisites"`
	ParallelSafe      bool     `json:"parallel_safe"`
	Status            string   `json:"status"`
	Ready             bool     `json:"ready"`
	AttemptOrdinal    uint32   `json:"attempt_ordinal"`
	ProviderBindingID string   `json:"provider_binding_id,omitempty"`
	EvidenceRecordIDs []string `json:"evidence_record_ids"`
	Outcome           string   `json:"outcome,omitempty"`
}

type TLC51MissionHumanWait struct {
	RequestID string `json:"request_id"`
	Boundary  string `json:"boundary"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
}

type TLC51MissionEffect struct {
	Effect               string    `json:"effect"`
	OperationID          string    `json:"operation_id"`
	IdempotencyKey       string    `json:"idempotency_key,omitempty"`
	ExternalState        string    `json:"external_state"`
	ReconciliationAction string    `json:"reconciliation_action,omitempty"`
	Outcome              string    `json:"outcome,omitempty"`
	ObservationID        string    `json:"observation_id,omitempty"`
	ObservedAt           time.Time `json:"observed_at,omitempty"`
}

type missionTLC51ProjectionCache struct {
	projection TLC51MissionControlEnvelope
	observedAt time.Time
	endpoint   string
	valid      bool
}

func (a *missionControlAcquirer) acquireTLC51(ctx context.Context, now time.Time) (*TLC51MissionControlEnvelope, MissionEvidenceMark, error) {
	base := strings.TrimSpace(os.Getenv("HIVE_OPS_API_BASE_URL"))
	endpoint := ""
	if base != "" {
		endpoint = strings.TrimRight(base, "/") + tlc51MissionControlPath
	}
	projection, err := a.fetchTLC51(ctx, endpoint, now)
	if err == nil {
		a.mu.Lock()
		a.tlc51 = missionTLC51ProjectionCache{projection: projection, observedAt: projection.GeneratedAt, endpoint: endpoint, valid: true}
		a.mu.Unlock()
		mark := missionSiteMark("current", "exact", "site_hive_tlc51_transport", projection.GeneratedAt, now, []string{"hive:factory-tlc51-mission-control"}, "")
		return &projection, mark, nil
	}
	a.mu.Lock()
	cached := a.tlc51
	a.mu.Unlock()
	reason := "Hive TLC 5.1 Mission Control acquisition failed; upstream details are withheld."
	if cached.valid && cached.endpoint == endpoint && !now.Before(cached.observedAt) && now.Sub(cached.observedAt) <= missionControlRetention {
		copy := cached.projection
		mark := missionSiteMark("stale", "exact", "site_hive_tlc51_transport", cached.observedAt, now, []string{"hive:factory-tlc51-mission-control"}, reason)
		return &copy, mark, errors.New(reason + "; retaining last schema-valid response")
	}
	return nil, missionSiteMark("unavailable", "unavailable", "site_hive_tlc51_transport", time.Time{}, now, nil, reason), errors.New(reason)
}

func (a *missionControlAcquirer) fetchTLC51(ctx context.Context, endpoint string, now time.Time) (TLC51MissionControlEnvelope, error) {
	if endpoint == "" {
		return TLC51MissionControlEnvelope{}, errors.New("HIVE_OPS_API_BASE_URL is not configured")
	}
	key := strings.TrimSpace(os.Getenv("HIVE_OPS_API_KEY"))
	if key == "" {
		return TLC51MissionControlEnvelope{}, errors.New("HIVE_OPS_API_KEY is required for the TLC 5.1 projection")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return TLC51MissionControlEnvelope{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := a.client.Do(req)
	if err != nil {
		return TLC51MissionControlEnvelope{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return TLC51MissionControlEnvelope{}, fmt.Errorf("Hive TLC 5.1 projection returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, missionControlMaxBytes+1))
	if err != nil {
		return TLC51MissionControlEnvelope{}, err
	}
	if len(body) > missionControlMaxBytes {
		return TLC51MissionControlEnvelope{}, errors.New("Hive TLC 5.1 projection exceeds limit")
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var projection TLC51MissionControlEnvelope
	if err := decoder.Decode(&projection); err != nil {
		return TLC51MissionControlEnvelope{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return TLC51MissionControlEnvelope{}, errors.New("Hive TLC 5.1 projection has trailing data")
	}
	if err := validateTLC51MissionControl(projection, now); err != nil {
		return TLC51MissionControlEnvelope{}, err
	}
	sort.Slice(projection.Orders, func(i, j int) bool {
		return projection.Orders[i].FactoryOrderID+"\x00"+projection.Orders[i].ChangeSeriesID < projection.Orders[j].FactoryOrderID+"\x00"+projection.Orders[j].ChangeSeriesID
	})
	return projection, nil
}

func validateTLC51MissionControl(projection TLC51MissionControlEnvelope, now time.Time) error {
	if projection.SchemaVersion != tlc51MissionControlSchema {
		return fmt.Errorf("unsupported TLC 5.1 Mission Control schema %q", projection.SchemaVersion)
	}
	if projection.GeneratedAt.IsZero() || projection.GeneratedAt.After(now.Add(5*time.Second)) {
		return errors.New("TLC 5.1 Mission Control generated_at is missing or in the future")
	}
	if projection.AuthorityGranted {
		return errors.New("TLC 5.1 Mission Control must not grant authority")
	}
	for index, item := range projection.Errors {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("TLC 5.1 projection error[%d] is empty", index)
		}
	}
	seen := map[string]struct{}{}
	for index, order := range projection.Orders {
		if order.SchemaVersion != "factory-tlc51-mission-control/v1" || order.ProtocolVersion != "factory-tlc51/v1" {
			return fmt.Errorf("TLC 5.1 order[%d] has unsupported schema or protocol", index)
		}
		if order.FactoryOrderID == "" || order.ChangeSeriesID == "" || order.OrderID == "" || order.OrderVersion == "" || order.Repository == "" ||
			!json.Valid(order.ReleaseIdentity) || !json.Valid(order.AdapterIdentity) || !json.Valid(order.Subject) ||
			!missionTLC51Digest(order.SubjectDigest) || !missionTLC51Digest(order.PlanDigest) {
			return fmt.Errorf("TLC 5.1 order[%d] has incomplete exact identity", index)
		}
		key := order.FactoryOrderID + "\x00" + order.ChangeSeriesID
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate TLC 5.1 FactoryOrder/change series %q", key)
		}
		seen[key] = struct{}{}
		if order.AuthorityGranted {
			return fmt.Errorf("TLC 5.1 order[%d] must not grant authority", index)
		}
		if order.InformationState != "CLASSIFIED" && order.InformationState != "UNCLASSIFIED" && order.InformationState != "BLOCKED_CONTRADICTION" {
			return fmt.Errorf("TLC 5.1 order[%d] has unknown information state", index)
		}
		if order.InformationState == "CLASSIFIED" && (order.Track == nil || !missionTLC51Track(*order.Track)) {
			return fmt.Errorf("TLC 5.1 order[%d] classified track is unavailable", index)
		}
		if order.Track != nil && !missionTLC51Track(*order.Track) {
			return fmt.Errorf("TLC 5.1 order[%d] has invalid track", index)
		}
		if order.RetainedFloor != nil && !missionTLC51Track(*order.RetainedFloor) {
			return fmt.Errorf("TLC 5.1 order[%d] has invalid retained floor", index)
		}
		if order.Decision != "pass" && order.Decision != "fail" && order.Decision != "unknown" {
			return fmt.Errorf("TLC 5.1 order[%d] has invalid decision", index)
		}
		if order.ReceiptDigest != "" && !missionTLC51Digest(order.ReceiptDigest) {
			return fmt.Errorf("TLC 5.1 order[%d] has invalid receipt digest", index)
		}
		if !order.GeneratedAt.Equal(projection.GeneratedAt) || order.EventGraphEventCount < 0 || order.WorkArtifactCount < 0 {
			return fmt.Errorf("TLC 5.1 order[%d] has invalid observation metadata", index)
		}
		if !missionTLC51Reconciliation(order.WorkReconciliation) || (order.WorkReconciliation == "match" && order.EventGraphEventCount != order.WorkArtifactCount) {
			return fmt.Errorf("TLC 5.1 order[%d] has invalid Work reconciliation", index)
		}
		obligations := map[string]struct{}{}
		for obligationIndex, obligation := range order.Obligations {
			if obligation.ID == "" || obligation.Kind == "" || obligation.Status == "" {
				return fmt.Errorf("TLC 5.1 order[%d] obligation[%d] has incomplete identity", index, obligationIndex)
			}
			if _, duplicate := obligations[obligation.ID]; duplicate {
				return fmt.Errorf("TLC 5.1 order[%d] has duplicate obligation %q", index, obligation.ID)
			}
			obligations[obligation.ID] = struct{}{}
		}
		for waitIndex, wait := range order.HumanWaits {
			if wait.RequestID == "" || wait.Boundary == "" || wait.Reason == "" || (wait.Status != "waiting" && wait.Status != "resolved") {
				return fmt.Errorf("TLC 5.1 order[%d] Human wait[%d] is invalid", index, waitIndex)
			}
		}
		for effectIndex, effect := range order.Effects {
			if effect.Effect == "" || effect.OperationID == "" || !missionTLC51ExternalState(effect.ExternalState) || (!effect.ObservedAt.IsZero() && effect.ObservedAt.After(now.Add(5*time.Second))) {
				return fmt.Errorf("TLC 5.1 order[%d] effect[%d] has invalid identity or observation time", index, effectIndex)
			}
		}
	}
	return nil
}

func missionTLC51Digest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if character < '0' || (character > '9' && character < 'a') || character > 'f' {
			return false
		}
	}
	return true
}

func missionTLC51Track(value string) bool {
	return value == "M" || value == "I" || value == "D" || value == "H"
}

func missionTLC51Reconciliation(value string) bool {
	switch value {
	case "match", "missing_both", "quarantine_missing_eventgraph", "repair_work_required", "quarantine_conflict", "quarantine_orphan_work", "unknown":
		return true
	default:
		return false
	}
}

func missionTLC51ExternalState(value string) bool {
	return value == "" || value == "absent" || value == "exact" || value == "conflict" || value == "unknown"
}

func missionTLC51Optional(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "unknown"
	}
	return *value
}

func missionTLC51Raw(value json.RawMessage) string {
	if len(value) == 0 {
		return "unknown"
	}
	return string(value)
}
