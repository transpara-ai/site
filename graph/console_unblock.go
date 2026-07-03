package graph

import (
	"fmt"
	"regexp"
	"strings"
)

// Change-control labels the hive intake gate reads (hive pkg/hive/issue_intake.go:18-21).
// Fixed constants: commands are built ONLY from these strings, never from
// projection free-text.
const (
	consoleLabelPRReady    = "cc:pr-ready"
	consoleLabelPRDeferred = "cc:pr-deferred"
	consoleLabelNeedsHuman = "cc:needs-human-scope"
	consoleLabelProtected  = "cc:protected-action"
)

// consoleUnblockScopeDenyLabels is the fixed removal order for the SCOPE
// command. cc:protected-action is deliberately absent: clearing a
// protected-action boundary is its own authority act and always renders as a
// separate command with warning copy (design D2, CFADA2-2).
var consoleUnblockScopeDenyLabels = []string{consoleLabelPRDeferred, consoleLabelNeedsHuman}

// consoleUnblockRepoPattern accepts only a plain GitHub owner/repo pair. The
// rendered command is operator-terminal copy-paste, so anything outside this
// character set (spaces, quotes, semicolons, redirects, HTML) refuses to render.
var consoleUnblockRepoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// consoleUnblockPlan is operator guidance derived purely from projected data.
// ScopeCommand and ProtectedCommand are split by authority class and are
// never merged: the protected-action removal must be a deliberate, separately
// copied act.
type consoleUnblockPlan struct {
	ScopeCommand     string
	ProtectedCommand string
	RescanCommand    string
}

type consoleLabelChip struct {
	Label string
	Kind  string // deny | ready | cc | plain
}

var consoleChipPattern = regexp.MustCompile(`^cc:[a-z0-9-]+$`)

// consoleIssueScanCardRef picks the ONE issue ref every unblock field reads
// from: target preferred, selected fallback — same preference as
// consoleIssueScanCardIssue (graph/console.go:227-233), never mixed.
func consoleIssueScanCardRef(card OpsCivilizationIssueScanKanbanCard) OpsCivilizationIssueRef {
	ref := card.TargetIssue
	if ref.Repo == "" && ref.Number == 0 {
		ref = card.SelectedIssue
	}
	return ref
}

// consoleIssueScanRunBlockerTypes indexes every blocker type projected for
// each RunID across the whole board. The unblock gate consults it so a card
// cannot offer a label command while a sibling card of the same run carries a
// blocker that labels cannot clear (CFADA1-1).
func consoleIssueScanRunBlockerTypes(board OpsCivilizationIssueScanKanban) map[string][]string {
	out := map[string][]string{}
	for _, col := range board.Columns {
		for _, card := range col.Cards {
			runID := strings.TrimSpace(card.RunID)
			if runID == "" {
				continue
			}
			for _, b := range card.Blockers {
				out[runID] = append(out[runID], strings.ToLower(strings.TrimSpace(b.BlockerType)))
			}
		}
	}
	return out
}

// consoleLabelSet normalizes exactly like hive's issueScanLabelSet
// (pkg/hive/issue_scan_parking.go:367-376), so the site's view of the label
// gate matches the gate itself.
func consoleLabelSet(labels []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, label := range labels {
		label = strings.ToLower(strings.TrimSpace(label))
		if label != "" {
			out[label] = struct{}{}
		}
	}
	return out
}

// consoleIssueScanUnblockPlan derives the exact operator commands that unblock
// a label-parked run — or refuses. Fail-closed allowlist: the permissive
// outcome (a command renders) is the explicitly-proven branch; every unknown,
// mismatched, unverifiable, or mixed state returns ok=false. There is no
// default that emits.
func consoleIssueScanUnblockPlan(card OpsCivilizationIssueScanKanbanCard, runBlockerTypes map[string][]string) (consoleUnblockPlan, bool) {
	runID := strings.TrimSpace(card.RunID)
	if runID == "" || len(card.Blockers) == 0 {
		return consoleUnblockPlan{}, false
	}
	types := runBlockerTypes[runID]
	if len(types) == 0 {
		return consoleUnblockPlan{}, false
	}
	ref := consoleIssueScanCardRef(card)
	if !consoleUnblockRepoPattern.MatchString(ref.Repo) || ref.Number <= 0 {
		return consoleUnblockPlan{}, false
	}
	labels := consoleLabelSet(ref.Labels)
	_, hasPRReady := labels[consoleLabelPRReady]
	_, hasDeferred := labels[consoleLabelPRDeferred]
	_, hasNeedsHuman := labels[consoleLabelNeedsHuman]
	_, hasProtected := labels[consoleLabelProtected]
	// Every blocker type projected for this run must be label-actionable AND
	// corroborated by the label evidence on the chosen ref (CFADA1-2). A
	// blocker without its evidence is a projection/label mismatch: refuse.
	for _, bt := range types {
		switch bt {
		case "needs_human_scope":
			if !hasDeferred && !hasNeedsHuman {
				return consoleUnblockPlan{}, false
			}
		case "protected_action":
			if !hasProtected {
				return consoleUnblockPlan{}, false
			}
		case "not_pr_ready":
			if hasPRReady {
				return consoleUnblockPlan{}, false
			}
		default:
			// stale_target, duplicate_chain, empty, or any future blocker
			// type: labels cannot clear it — no command.
			return consoleUnblockPlan{}, false
		}
	}
	base := fmt.Sprintf("gh issue edit %d --repo %s", ref.Number, ref.Repo)
	scopeParts := []string{base}
	for _, deny := range consoleUnblockScopeDenyLabels {
		if _, ok := labels[deny]; ok {
			scopeParts = append(scopeParts, "--remove-label "+deny)
		}
	}
	if !hasPRReady {
		scopeParts = append(scopeParts, "--add-label "+consoleLabelPRReady)
	}
	scope := ""
	if len(scopeParts) > 1 {
		scope = strings.Join(scopeParts, " ")
	}
	protected := ""
	if hasProtected {
		protected = base + " --remove-label " + consoleLabelProtected
	}
	if scope == "" && protected == "" {
		// Nothing to change: no deny labels present and pr-ready already set.
		return consoleUnblockPlan{}, false
	}
	return consoleUnblockPlan{
		ScopeCommand:     scope,
		ProtectedCommand: protected,
		RescanCommand:    "hive factory scan-issues --human YOUR_NAME --repo " + ref.Repo,
	}, true
}

// consoleIssueScanRefChips classifies the chosen ref's labels for display.
// Classification is an allowlist; everything unrecognized renders as plain
// escaped text (Kind "plain"), never dropped and never trusted.
func consoleIssueScanRefChips(card OpsCivilizationIssueScanKanbanCard) []consoleLabelChip {
	ref := consoleIssueScanCardRef(card)
	chips := make([]consoleLabelChip, 0, len(ref.Labels))
	for _, raw := range ref.Labels {
		display := strings.TrimSpace(raw)
		if display == "" {
			continue
		}
		normalized := strings.ToLower(display)
		var kind string
		switch {
		case normalized == consoleLabelPRReady:
			kind = "ready"
		case normalized == consoleLabelPRDeferred || normalized == consoleLabelNeedsHuman || normalized == consoleLabelProtected:
			kind = "deny"
		case consoleChipPattern.MatchString(normalized):
			kind = "cc"
		default:
			kind = "plain"
		}
		chips = append(chips, consoleLabelChip{Label: display, Kind: kind})
	}
	return chips
}
