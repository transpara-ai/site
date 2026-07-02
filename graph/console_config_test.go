package graph

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// testConfigModelSelection returns a populated model-selection projection:
// two catalog models and two role assignments with distinct, deterministic
// provenance paths (policy-event override vs plain-model inferred manual).
func testConfigModelSelection() OpsHiveModelSelection {
	return OpsHiveModelSelection{
		Source:        "hive-operator-projection",
		CatalogSource: "catalog-mixed.yaml",
		Models: []OpsHiveModelCatalogEntry{
			{ID: "claude-opus-4-6", Provider: "claude-cli", AuthMode: "subscription", Tier: "judgment"},
			{ID: "gpt-5.5", Provider: "codex-cli", AuthMode: "subscription", Tier: "execution"},
		},
		Assignments: []OpsHiveModelRoleAssignment{
			// Policy-event assignment → obsAssignmentModelModeState = ("Manual", "override").
			{Role: "strategist", Model: "claude-opus-4-6", Provider: "claude-cli", Source: "hive-model-policy-event", PolicyEventID: "evt_123"},
			// Plain resolved model, no mode metadata → ("Manual", "inferred").
			{Role: "implementer", Model: "gpt-5.5", Provider: "codex-cli"},
		},
	}
}

func TestBuildConsoleConfig(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)

	t.Run("fetch error is unavailable with notice and no invented routing", func(t *testing.T) {
		cfg := buildConsoleConfig(nil, errors.New("HIVE_OPS_API_BASE_URL is not configured"), now)
		if cfg.Freshness != FreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 {
			t.Fatal("expected a notice explaining the unavailable state")
		}
		if len(cfg.Selection.Models) != 0 || len(cfg.Selection.Assignments) != 0 {
			t.Fatal("unavailable config must not invent catalog or assignments")
		}
	})

	t.Run("nil projection is unavailable", func(t *testing.T) {
		cfg := buildConsoleConfig(nil, nil, now)
		if cfg.Freshness != FreshnessUnavailable || len(cfg.Notices) == 0 {
			t.Fatalf("nil projection: freshness = %q, notices = %v; want unavailable with notice", cfg.Freshness, cfg.Notices)
		}
	})

	t.Run("fresh projection with EMPTY model selection fails closed", func(t *testing.T) {
		// A fresh timestamp with no model-selection data must NOT render as a
		// current-but-empty surface; only a selection carrying data earns rendering.
		proj := &OpsHiveProjection{GeneratedAt: now.Add(-2 * time.Second).Format(time.RFC3339)}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable for empty selection", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 || !strings.Contains(strings.Join(cfg.Notices, "; "), "no model-selection") {
			t.Fatalf("empty selection must carry an explicit notice, got %v", cfg.Notices)
		}
	})

	t.Run("populated fresh projection is current and passes selection through", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Second).Format(time.RFC3339),
			ModelSelection: testConfigModelSelection(),
		}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessCurrent {
			t.Fatalf("freshness = %q, want current", cfg.Freshness)
		}
		if len(cfg.Selection.Models) != 2 || len(cfg.Selection.Assignments) != 2 {
			t.Fatalf("selection not passed through: %d models, %d assignments", len(cfg.Selection.Models), len(cfg.Selection.Assignments))
		}
	})

	t.Run("stale timestamp is stale", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Minute).Format(time.RFC3339), // older than consoleStaleWindow (30s)
			ModelSelection: testConfigModelSelection(),
		}
		if cfg := buildConsoleConfig(proj, nil, now); cfg.Freshness != FreshnessStale {
			t.Fatalf("freshness = %q, want stale", cfg.Freshness)
		}
	})

	t.Run("projection errors downgrade fresh data to partial", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Second).Format(time.RFC3339),
			ModelSelection: testConfigModelSelection(),
			Errors:         []string{"telemetry source degraded"},
		}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessPartial {
			t.Fatalf("freshness = %q, want partial", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 || cfg.Notices[0] != "telemetry source degraded" {
			t.Fatalf("projection errors must surface as notices, got %v", cfg.Notices)
		}
	})

	t.Run("model-selection errors also downgrade to partial", func(t *testing.T) {
		sel := testConfigModelSelection()
		sel.Errors = []string{"catalog reload failed"}
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Second).Format(time.RFC3339),
			ModelSelection: sel,
		}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessPartial {
			t.Fatalf("freshness = %q, want partial for selection errors", cfg.Freshness)
		}
	})

	t.Run("populated selection with missing timestamp is unavailable with notice", func(t *testing.T) {
		proj := &OpsHiveProjection{ModelSelection: testConfigModelSelection()} // GeneratedAt empty
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable for missing timestamp", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 {
			t.Fatal("timestamp-less unavailable state must carry an explicit notice")
		}
	})
}
