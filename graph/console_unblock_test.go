package graph

import "testing"

func unblockCard(runID, blockerType string, labels []string) OpsCivilizationIssueScanKanbanCard {
	return OpsCivilizationIssueScanKanbanCard{
		RunID:       runID,
		Blockers:    []OpsCivilizationIssueScanBlockerProjected{{BlockerType: blockerType, RequiredAction: "projected action"}},
		TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 226, Labels: labels},
	}
}

func boardWith(cards ...OpsCivilizationIssueScanKanbanCard) OpsCivilizationIssueScanKanban {
	return OpsCivilizationIssueScanKanban{Columns: []OpsCivilizationIssueScanKanbanColumn{{State: "blocked", Label: "Human action", Cards: cards}}}
}

// TestConsoleUnblockPlanDomain is the fix-the-class test: it walks the whole
// blocker-type input domain (all five known types, empty, and a future value)
// crossed with label evidence, repo validity, and number validity. The
// permissive outcome (a command renders) must be the explicitly-proven branch.
func TestConsoleUnblockPlanDomain(t *testing.T) {
	cases := []struct {
		name          string
		card          OpsCivilizationIssueScanKanbanCard
		siblings      []OpsCivilizationIssueScanKanbanCard
		wantOK        bool
		wantScope     string
		wantProtected string
		wantRescan    string
	}{
		{
			name:       "needs_human_scope with scope deny labels only",
			card:       unblockCard("run-1", "needs_human_scope", []string{"cc:needs-human-scope", "cc:pr-deferred", "cc:intake"}),
			wantOK:     true,
			wantScope:  "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:pr-deferred --remove-label cc:needs-human-scope --add-label cc:pr-ready",
			wantRescan: "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			// The live docs#226 shape: one witnessed blocker (first-match
			// parking), full deny-label superset. Protected removal must be
			// its OWN command, never folded into the scope command.
			name:          "label superset splits protected removal into separate command",
			card:          unblockCard("run-2", "needs_human_scope", []string{"cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action"}),
			wantOK:        true,
			wantScope:     "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:pr-deferred --remove-label cc:needs-human-scope --add-label cc:pr-ready",
			wantProtected: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action",
			wantRescan:    "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			name:          "protected_action corroborated",
			card:          unblockCard("run-3", "protected_action", []string{"cc:protected-action"}),
			wantOK:        true,
			wantScope:     "gh issue edit 226 --repo transpara-ai/docs --add-label cc:pr-ready",
			wantProtected: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action",
			wantRescan:    "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			// codex CFADA round-2 scenario: run projects only not_pr_ready but
			// the ref carries the protected label. The protected boundary may
			// only be cleared via the separate, warned command.
			name:          "not_pr_ready with protected label present keeps commands split",
			card:          unblockCard("run-4", "not_pr_ready", []string{"cc:protected-action"}),
			wantOK:        true,
			wantScope:     "gh issue edit 226 --repo transpara-ai/docs --add-label cc:pr-ready",
			wantProtected: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action",
			wantRescan:    "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			name:       "not_pr_ready with no labels renders add-only scope command",
			card:       unblockCard("run-5", "not_pr_ready", nil),
			wantOK:     true,
			wantScope:  "gh issue edit 226 --repo transpara-ai/docs --add-label cc:pr-ready",
			wantRescan: "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			name:       "label normalization matches hive (case + whitespace)",
			card:       unblockCard("run-6", "needs_human_scope", []string{"  CC:Needs-Human-Scope  "}),
			wantOK:     true,
			wantScope:  "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:needs-human-scope --add-label cc:pr-ready",
			wantRescan: "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		// ---- everything below MUST fail closed ----
		{name: "stale_target never gets a command", card: unblockCard("run-5", "stale_target", []string{"cc:needs-human-scope"})},
		{name: "duplicate_chain never gets a command", card: unblockCard("run-6", "duplicate_chain", nil)},
		{name: "empty blocker type", card: unblockCard("run-7", "", []string{"cc:needs-human-scope"})},
		{name: "future blocker type", card: unblockCard("run-8", "future_blocker_v2", []string{"cc:needs-human-scope"})},
		{name: "no blockers at all", card: OpsCivilizationIssueScanKanbanCard{RunID: "run-9", TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 226}}},
		{name: "empty run id cannot be verified run-wide", card: unblockCard("", "needs_human_scope", []string{"cc:needs-human-scope"})},
		{
			name:     "sibling card on same run carries non-label blocker",
			card:     unblockCard("run-10", "needs_human_scope", []string{"cc:needs-human-scope"}),
			siblings: []OpsCivilizationIssueScanKanbanCard{unblockCard("run-10", "stale_target", nil)},
		},
		{
			name:     "sibling card on same run carries unknown blocker",
			card:     unblockCard("run-11", "protected_action", []string{"cc:protected-action"}),
			siblings: []OpsCivilizationIssueScanKanbanCard{unblockCard("run-11", "mystery_blocker", nil)},
		},
		{name: "needs_human_scope without corroborating deny label (mismatch)", card: unblockCard("run-12", "needs_human_scope", []string{"cc:intake"})},
		{name: "protected_action without cc:protected-action label (mismatch)", card: unblockCard("run-13", "protected_action", []string{"cc:pr-deferred"})},
		{name: "not_pr_ready but cc:pr-ready already present (mismatch)", card: unblockCard("run-14", "not_pr_ready", []string{"cc:pr-ready"})},
		{name: "invalid repo missing slash", card: func() OpsCivilizationIssueScanKanbanCard {
			c := unblockCard("run-15", "not_pr_ready", nil)
			c.TargetIssue.Repo = "docs"
			return c
		}()},
		{name: "hostile repo with shell metacharacters", card: func() OpsCivilizationIssueScanKanbanCard {
			c := unblockCard("run-16", "not_pr_ready", nil)
			c.TargetIssue.Repo = "transpara-ai/docs; rm -rf /"
			return c
		}()},
		{name: "hostile repo with html", card: func() OpsCivilizationIssueScanKanbanCard {
			c := unblockCard("run-17", "not_pr_ready", nil)
			c.TargetIssue.Repo = "transpara-ai/docs\" onmouseover=\"x"
			return c
		}()},
		{name: "zero issue number", card: func() OpsCivilizationIssueScanKanbanCard {
			c := unblockCard("run-18", "not_pr_ready", nil)
			c.TargetIssue.Number = 0
			return c
		}()},
		{name: "negative issue number", card: func() OpsCivilizationIssueScanKanbanCard {
			c := unblockCard("run-19", "not_pr_ready", nil)
			c.TargetIssue.Number = -4
			return c
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			board := boardWith(append([]OpsCivilizationIssueScanKanbanCard{tc.card}, tc.siblings...)...)
			plan, ok := consoleIssueScanUnblockPlan(tc.card, consoleIssueScanRunBlockerTypes(board))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (plan %+v)", ok, tc.wantOK, plan)
			}
			if !ok {
				if plan.ScopeCommand != "" || plan.ProtectedCommand != "" || plan.RescanCommand != "" {
					t.Fatalf("fail-closed plan must be empty, got %+v", plan)
				}
				return
			}
			if plan.ScopeCommand != tc.wantScope {
				t.Errorf("scope command\n got: %s\nwant: %s", plan.ScopeCommand, tc.wantScope)
			}
			if plan.ProtectedCommand != tc.wantProtected {
				t.Errorf("protected command\n got: %s\nwant: %s", plan.ProtectedCommand, tc.wantProtected)
			}
			if plan.RescanCommand != tc.wantRescan {
				t.Errorf("rescan command\n got: %s\nwant: %s", plan.RescanCommand, tc.wantRescan)
			}
		})
	}
}

// The plan must read every field from ONE ref: target preferred, selected as
// fallback — never mixed.
func TestConsoleUnblockPlanUsesSelectedRefWhenTargetEmpty(t *testing.T) {
	card := OpsCivilizationIssueScanKanbanCard{
		RunID:         "run-20",
		Blockers:      []OpsCivilizationIssueScanBlockerProjected{{BlockerType: "not_pr_ready"}},
		SelectedIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/work", Number: 55},
	}
	plan, ok := consoleIssueScanUnblockPlan(card, consoleIssueScanRunBlockerTypes(boardWith(card)))
	if !ok {
		t.Fatal("expected plan from selected-issue fallback")
	}
	want := "gh issue edit 55 --repo transpara-ai/work --add-label cc:pr-ready"
	if plan.ScopeCommand != want {
		t.Fatalf("scope command = %q, want %q", plan.ScopeCommand, want)
	}
	if plan.ProtectedCommand != "" {
		t.Fatalf("no protected label present; protected command must be empty, got %q", plan.ProtectedCommand)
	}
}

func TestConsoleIssueScanRefChips(t *testing.T) {
	card := unblockCard("run-21", "needs_human_scope", []string{"cc:needs-human-scope", "cc:pr-ready", "cc:intake", "Weird Label!"})
	chips := consoleIssueScanRefChips(card)
	want := []consoleLabelChip{
		{Label: "cc:needs-human-scope", Kind: "deny"},
		{Label: "cc:pr-ready", Kind: "ready"},
		{Label: "cc:intake", Kind: "cc"},
		{Label: "Weird Label!", Kind: "plain"},
	}
	if len(chips) != len(want) {
		t.Fatalf("chips = %+v, want %+v", chips, want)
	}
	for i := range want {
		if chips[i] != want[i] {
			t.Errorf("chip[%d] = %+v, want %+v", i, chips[i], want[i])
		}
	}
}
