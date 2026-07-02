package graph

import (
	"strings"
	"time"
)

// ConsoleConfig is the Config-tab view-model: the hive model-routing
// projection (catalog + per-role assignments + provenance) plus an explicit
// freshness state. It fails closed — a nil, failed, empty, or timestamp-less
// projection renders as unavailable with a human-readable notice and no
// invented routing data.
type ConsoleConfig struct {
	Freshness   ConsoleFreshness
	GeneratedAt string
	Selection   OpsHiveModelSelection
	Notices     []string
}

// buildConsoleConfig maps the hive operator projection (or a fetch error)
// into the Config view-model. Fail-closed by allowlist: only a model
// selection carrying actual data (source, models, or assignments) renders as
// data; an empty selection — even on a fresh projection — resolves to
// unavailable rather than a current-but-empty surface that could read as
// "hive routes nothing".
func buildConsoleConfig(proj *OpsHiveProjection, fetchErr error, now time.Time) ConsoleConfig {
	if fetchErr != nil || proj == nil {
		reason := "operator projection unavailable"
		if fetchErr != nil {
			reason = fetchErr.Error()
		}
		return ConsoleConfig{Freshness: FreshnessUnavailable, Notices: []string{reason}}
	}
	sel := proj.ModelSelection
	notices := append([]string(nil), proj.Errors...)
	notices = append(notices, sel.Errors...)
	if sel.Source == "" && len(sel.Models) == 0 && len(sel.Assignments) == 0 {
		return ConsoleConfig{
			Freshness: FreshnessUnavailable,
			Notices:   append(notices, "hive operator projection carries no model-selection data"),
		}
	}
	freshness := deriveFreshness(proj.GeneratedAt, nil, len(notices) > 0, now, consoleStaleWindow)
	if freshness == FreshnessUnavailable && len(notices) == 0 {
		notices = []string{"operator projection timestamp missing or unparseable"}
	}
	return ConsoleConfig{
		Freshness:   freshness,
		GeneratedAt: proj.GeneratedAt,
		Selection:   sel,
		Notices:     notices,
	}
}

// consoleConfigAssignmentModel prefers the resolved model, then the policy
// model, then an honest placeholder — never an empty cell that reads as data.
func consoleConfigAssignmentModel(item OpsHiveModelRoleAssignment) string {
	if m := strings.TrimSpace(item.Model); m != "" {
		return m
	}
	if m := strings.TrimSpace(item.PolicyModel); m != "" {
		return m
	}
	return "not projected"
}

func consoleConfigAssignmentProvider(item OpsHiveModelRoleAssignment) string {
	if p := strings.TrimSpace(item.Provider); p != "" {
		return p
	}
	if p := strings.TrimSpace(item.PolicyProvider); p != "" {
		return p
	}
	return "not projected"
}

// consoleConfigAssignmentMode renders "Mode · provenance" for a role card,
// reusing the /ops observatory derivation verbatim so /console and /ops can
// never disagree about why a role routes where it does.
func consoleConfigAssignmentMode(sel OpsHiveModelSelection, item OpsHiveModelRoleAssignment) string {
	mode, provenance := obsAssignmentModelModeState(sel, item)
	return mode + " · " + provenance
}

func consoleConfigGlobalMode(sel OpsHiveModelSelection) string {
	mode, provenance, _ := obsHiveProjectionModelModeState(sel)
	return mode + " · " + provenance
}
