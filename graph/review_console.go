package graph

import (
	"html/template"
	"net/http"
	"strings"
)

type OpsReviewConsoleData struct {
	Title               string
	GeneratedAt         string
	AuthorizationSource string
	Boundary            []string
	ExactHeadEvidence   []OpsExactHeadApprovalEvidence
	DecisionQueue       []OpsExternalCommitteeDecisionQueueItem
	Items               []OpsReviewItem
}

type OpsExactHeadApprovalEvidence struct {
	ID                string
	Title             string
	TargetRepo        string
	TargetRef         string
	RequiredHead      string
	ApprovedHead      string
	ApprovalSourceURL string
	ApprovalActor     string
	ApprovalState     string
	ResidualState     string
	CleanApproval     bool
	Summary           string
	Limitation        string
	DisplayOnly       bool
}

type OpsExternalCommitteeDecisionQueueItem struct {
	ID             string
	Title          string
	SourceIssue    string
	SourceURL      string
	SourceRepo     string
	QueueState     string
	EvidenceState  string
	RequiredActor  string
	DecisionNeeded string
	Blocker        string
	NoWriteLimit   string
	DisplayOnly    bool
}

type OpsReviewItem struct {
	ID             string
	Title          string
	DecisionKind   string
	SourceURL      string
	SourceType     string
	SourceRepo     string
	ExactHead      string
	RequiredActor  string
	RequiredAction string
	EvidenceState  string
	ResidualState  string
	GateScope      string
	Limitation     string
	DisplayOnly    bool
}

func (h *Handlers) handleOpsReviewConsole(w http.ResponseWriter, r *http.Request) {
	data := buildOpsReviewConsoleData()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := opsReviewConsoleTemplate.Execute(w, data); err != nil {
		http.Error(w, "render review console: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func buildOpsReviewConsoleData() OpsReviewConsoleData {
	return OpsReviewConsoleData{
		Title:               "External Committee Review Console",
		GeneratedAt:         "static Event 13 fixture; not a live recency or revocation check",
		AuthorizationSource: "docs v4.0 Event 13 AuthorityDecision, merged by docs#185",
		Boundary: []string{
			"display_only=true for every review item",
			"no approve, reject, merge, close, label, comment, deploy, RuntimeBroker, EventGraph write, or protected action path",
			"Site is a console, not the truth source",
			"Gate W remains open until a later docs evidence-decision PR accepts, rejects, or defers the Site implementation evidence",
		},
		ExactHeadEvidence: []OpsExactHeadApprovalEvidence{
			buildOpsExactHeadApprovalEvidence(opsExactHeadApprovalFixture{
				ID:                "site-123-approved",
				Title:             "Site #123 CFAR approval evidence",
				TargetRepo:        "transpara-ai/site",
				TargetRef:         "pull/123",
				RequiredHead:      "d1ca6cd93f87072d4422b9de1e8574f4bfa973a9",
				ApprovedHead:      "d1ca6cd93f87072d4422b9de1e8574f4bfa973a9",
				ApprovalSourceURL: "https://github.com/transpara-ai/site/pull/123#issuecomment-4801761943",
				ApprovalActor:     "CFAR",
			}),
			buildOpsExactHeadApprovalEvidence(opsExactHeadApprovalFixture{
				ID:                "site-stale-head-fixture",
				Title:             "Stale exact-head approval fixture",
				TargetRepo:        "transpara-ai/site",
				TargetRef:         "pull/example-stale",
				RequiredHead:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				ApprovedHead:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				ApprovalSourceURL: "https://github.com/transpara-ai/site/issues/118",
				ApprovalActor:     "External Committee",
			}),
			buildOpsExactHeadApprovalEvidence(opsExactHeadApprovalFixture{
				ID:           "gate-w-missing-fixture",
				Title:        "Missing exact-head approval fixture",
				TargetRepo:   "transpara-ai/docs",
				TargetRef:    "gate-w-closeout",
				RequiredHead: "cccccccccccccccccccccccccccccccccccccccc",
			}),
			buildOpsExactHeadApprovalEvidence(opsExactHeadApprovalFixture{
				ID:                "docs-172-accepted-residual",
				Title:             "Gate S exact-head approval with accepted residual",
				TargetRepo:        "transpara-ai/docs",
				TargetRef:         "issues/172",
				RequiredHead:      "dddddddddddddddddddddddddddddddddddddddd",
				ApprovedHead:      "dddddddddddddddddddddddddddddddddddddddd",
				ApprovalSourceURL: "https://github.com/transpara-ai/docs/issues/172#issuecomment-4757178868",
				ApprovalActor:     "External Committee",
				ResidualState:     "accepted_with_residual",
			}),
		},
		DecisionQueue: []OpsExternalCommitteeDecisionQueueItem{
			{
				ID:             "site-117-human-required-decision",
				Title:          "External Committee issue decision queue display",
				SourceIssue:    "site#117",
				SourceURL:      "https://github.com/transpara-ai/site/issues/117",
				SourceRepo:     "transpara-ai/site",
				QueueState:     "human_required",
				EvidenceState:  "current",
				RequiredActor:  "External Committee",
				DecisionNeeded: "Review issue-sourced protected-action blockers and decide whether a later authority packet is needed.",
				Blocker:        "This Site surface is read-only and cannot approve, deny, merge, label, comment, deploy, or close the issue.",
				NoWriteLimit:   opsReviewConsoleNoWriteLimit(),
				DisplayOnly:    true,
			},
			{
				ID:             "site-116-missing-inputs",
				Title:          "Replace temporary Civilization fixture display with issue and EventGraph projection inputs",
				SourceIssue:    "site#116",
				SourceURL:      "https://github.com/transpara-ai/site/issues/116",
				SourceRepo:     "transpara-ai/site",
				QueueState:     "blocked_missing_evidence",
				EvidenceState:  "missing",
				RequiredActor:  "External Committee",
				DecisionNeeded: "Require EventGraph and issue read-input evidence before this can move from fixture display to live projection input.",
				Blocker:        "Missing read input evidence must remain blocked and cannot be treated as approval.",
				NoWriteLimit:   opsReviewConsoleNoWriteLimit(),
				DisplayOnly:    true,
			},
			{
				ID:             "site-118-stale-evidence",
				Title:          "Stale exact-head approval evidence",
				SourceIssue:    "site#118",
				SourceURL:      "https://github.com/transpara-ai/site/issues/118",
				SourceRepo:     "transpara-ai/site",
				QueueState:     "blocked_stale_evidence",
				EvidenceState:  "stale",
				RequiredActor:  "External Committee",
				DecisionNeeded: "Refresh exact-head approval evidence before citing the issue as clean approval.",
				Blocker:        "Stale evidence fails closed and cannot unlock protected actions.",
				NoWriteLimit:   opsReviewConsoleNoWriteLimit(),
				DisplayOnly:    true,
			},
			{
				ID:             "docs-172-residual-evidence",
				Title:          "Docs #172 residual approval evidence",
				SourceIssue:    "docs#172",
				SourceURL:      "https://github.com/transpara-ai/docs/issues/172",
				SourceRepo:     "transpara-ai/docs",
				QueueState:     "blocked_residual_evidence",
				EvidenceState:  "accepted_residual",
				RequiredActor:  "External Committee",
				DecisionNeeded: "Refresh or carry exact-head approval evidence before citing this issue as clean approval.",
				Blocker:        "Accepted residual evidence must be carried and cannot unlock protected actions as clean approval.",
				NoWriteLimit:   opsReviewConsoleNoWriteLimit(),
				DisplayOnly:    true,
			},
		},
		Items: []OpsReviewItem{
			{
				ID:             "docs-185-exact-head-approval",
				Title:          "Event 13 AuthorityDecision exact-head approval",
				DecisionKind:   "exact_head_approval",
				SourceURL:      "https://github.com/transpara-ai/docs/pull/185#issuecomment-4767206936",
				SourceType:     "pull_request",
				SourceRepo:     "transpara-ai/docs",
				ExactHead:      "127da4ef57dee34231cc50d87a249349fc0f768c",
				RequiredActor:  "External Committee",
				RequiredAction: "approve exact head before merge",
				EvidenceState:  "passed",
				ResidualState:  "none",
				GateScope:      "Event 13 / Gate W authority packet",
				Limitation:     "Passed reflects a verified point-in-time docs#185 approval for this pinned head; this static console is not a live revocation or staleness checker and does not close Gate W.",
				DisplayOnly:    true,
			},
			{
				ID:             "event-13-authority-decision",
				Title:          "Event 13 AuthorityDecision",
				DecisionKind:   "authority_decision",
				SourceURL:      "https://github.com/transpara-ai/docs/blob/main/dark-factory/v4.0/implementation/epics/epic-13-external-committee-review-console/03-external-committee-review-console-authority-decision-v4.0.md",
				SourceType:     "docs_packet",
				SourceRepo:     "transpara-ai/docs",
				RequiredActor:  "operator and External Committee",
				RequiredAction: "respect one bounded Site PR lifecycle and stop conditions",
				EvidenceState:  "passed",
				ResidualState:  "none",
				GateScope:      "DF-V4.0-EPIC-013-AUTHORITY-DECISION",
				Limitation:     "The AuthorityDecision authorizes only this Level 0 display implementation lifecycle; it grants no protected action, runtime, deploy, or closure authority.",
				DisplayOnly:    true,
			},
			{
				ID:             "docs-172-gate-s-residual",
				Title:          "Gate S approval artifact residual",
				DecisionKind:   "residual_disposition",
				SourceURL:      "https://github.com/transpara-ai/docs/issues/172#issuecomment-4757178868",
				SourceType:     "issue",
				SourceRepo:     "transpara-ai/docs",
				RequiredActor:  "External Committee",
				RequiredAction: "carry residual with later citations",
				EvidenceState:  "carried_residual",
				ResidualState:  "accepted_with_residual",
				GateScope:      "Event 9 / Gate S closeout",
				Limitation:     "The merge event can be cited only with the residual attached; it is not clean standalone exact-head approval evidence.",
				DisplayOnly:    true,
			},
			{
				ID:             "gate-w-closeout",
				Title:          "Gate W closeout evidence",
				DecisionKind:   "gate_closeout",
				SourceURL:      "https://github.com/transpara-ai/docs/pull/185",
				SourceType:     "docs_packet",
				SourceRepo:     "transpara-ai/docs",
				RequiredActor:  "operator and External Committee",
				RequiredAction: "merge bounded Site PR, then accept/reject/defer evidence in later docs PR",
				EvidenceState:  "missing",
				ResidualState:  "open",
				GateScope:      "Event 13 / Gate W",
				Limitation:     "This Site page cannot self-close Gate W and cannot treat missing implementation evidence as accepted.",
				DisplayOnly:    true,
			},
			{
				ID:             "test-001-yellow",
				Title:          "Test 001 live evidence tracker",
				DecisionKind:   "issue_disposition",
				SourceURL:      "https://github.com/transpara-ai/operation/issues/26",
				SourceType:     "issue",
				SourceRepo:     "transpara-ai/operation",
				RequiredActor:  "operator and External Committee",
				RequiredAction: "record durable live evidence or accepted missing-evidence findings",
				EvidenceState:  "pending",
				ResidualState:  "open",
				GateScope:      "Test 001 remains YELLOW",
				Limitation:     "Progress-display, local-render, and review-console evidence do not prove production runtime, live-reader proof, or public-correction proof.",
				DisplayOnly:    true,
			},
		},
	}
}

func opsReviewConsoleNoWriteLimit() string {
	return "Display only: no approve, deny, merge, label, comment, GitHub write, Hive write, EventGraph write, runtime execution, deploy, gate closure, autonomy increase, value allocation, or protected-action approval."
}

type opsExactHeadApprovalFixture struct {
	ID                string
	Title             string
	TargetRepo        string
	TargetRef         string
	RequiredHead      string
	ApprovedHead      string
	ApprovalSourceURL string
	ApprovalActor     string
	ResidualState     string
}

func buildOpsExactHeadApprovalEvidence(f opsExactHeadApprovalFixture) OpsExactHeadApprovalEvidence {
	state := opsExactHeadApprovalState(f.RequiredHead, f.ApprovedHead, f.ApprovalSourceURL, f.ResidualState)
	return OpsExactHeadApprovalEvidence{
		ID:                f.ID,
		Title:             f.Title,
		TargetRepo:        f.TargetRepo,
		TargetRef:         f.TargetRef,
		RequiredHead:      f.RequiredHead,
		ApprovedHead:      f.ApprovedHead,
		ApprovalSourceURL: f.ApprovalSourceURL,
		ApprovalActor:     f.ApprovalActor,
		ApprovalState:     state,
		ResidualState:     opsReviewResidualState(f.ResidualState),
		CleanApproval:     state == "approved",
		Summary:           opsExactHeadApprovalSummary(state),
		Limitation:        opsExactHeadApprovalLimitation(state),
		DisplayOnly:       true,
	}
}

func opsExactHeadApprovalState(requiredHead, approvedHead, approvalSourceURL, residualState string) string {
	requiredHead = strings.TrimSpace(requiredHead)
	approvedHead = strings.TrimSpace(approvedHead)
	approvalSourceURL = strings.TrimSpace(approvalSourceURL)
	residualState = opsReviewResidualState(residualState)
	if requiredHead == "" || approvedHead == "" || approvalSourceURL == "" {
		return "missing"
	}
	// Review Console fixtures use full commit SHAs. Any mismatch fails closed as stale.
	if approvedHead != requiredHead {
		return "stale"
	}
	if residualState != "none" {
		return "accepted_residual"
	}
	return "approved"
}

func opsReviewResidualState(state string) string {
	state = strings.TrimSpace(state)
	if state == "" {
		return "none"
	}
	return state
}

func opsExactHeadApprovalSummary(state string) string {
	switch state {
	case "approved":
		return "Exact-head approval evidence matches the required head."
	case "stale":
		return "Approval evidence exists for a different head; the target must be re-approved at the current head."
	case "accepted_residual":
		return "Exact-head approval evidence exists, but only with an accepted residual attached."
	default:
		return "Exact-head approval evidence is missing."
	}
}

func opsExactHeadApprovalLimitation(state string) string {
	switch state {
	case "approved":
		return "Clean approval is display evidence only; Site does not merge, deploy, close gates, or approve protected actions."
	case "stale":
		return "Stale exact-head approval is not approval for the current head."
	case "accepted_residual":
		return "Accepted residual evidence must be carried with later citations and is not clean approval."
	default:
		return "Missing evidence fails closed and is not approval."
	}
}

var opsReviewConsoleTemplate = template.Must(template.New("ops-review-console").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>{{.Title}}</title>
	<link rel="stylesheet" href="/static/css/site.css">
</head>
<body class="bg-void text-warm-secondary min-h-screen">
	<main class="max-w-6xl mx-auto px-4 md:px-6 py-8 md:py-10 space-y-6">
		<section class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
			<div class="space-y-2 max-w-3xl">
				<p class="text-xs font-medium text-brand uppercase tracking-widest">Operator shell</p>
				<h1 class="text-3xl md:text-4xl font-display font-normal text-warm">{{.Title}}</h1>
				<p class="text-sm md:text-base text-warm-muted leading-relaxed">Read-only Event 13 decision evidence surface for exact-head approvals, process residuals, authority packets, and gate closeout state.</p>
				<p class="text-xs text-warm-faint">Authorization: {{.AuthorizationSource}} · rendered: {{.GeneratedAt}}</p>
			</div>
			<a href="/ops" class="inline-flex items-center justify-center px-3 py-2 rounded-md bg-elevated text-warm-muted hover:text-brand transition-colors text-sm">Operations</a>
		</section>

		<section class="grid gap-3 md:grid-cols-2 lg:grid-cols-4" data-review-console-boundary="display-only">
			{{range .Boundary}}
				<div class="border border-edge bg-surface rounded-lg p-4 min-h-[7rem]">
					<p class="text-xs text-warm-muted leading-relaxed">{{.}}</p>
				</div>
			{{end}}
		</section>

		<section class="border border-edge bg-surface rounded-lg overflow-hidden" data-exact-head-approval-evidence="read-only">
			<header class="px-4 py-3 border-b border-edge">
				<h2 class="text-sm font-medium text-warm">Exact-head approval evidence</h2>
				<p class="text-xs text-warm-faint mt-1">Read-only issue and PR evidence fixtures. Missing, stale, or residual-carrying evidence fails closed and is never shown as clean approval.</p>
			</header>
			<div class="divide-y divide-edge">
				{{range .ExactHeadEvidence}}
					<article class="p-4 space-y-3" data-exact-head-evidence="{{.ID}}" data-approval-state="{{.ApprovalState}}" data-clean-approval="{{.CleanApproval}}" data-display-only="{{.DisplayOnly}}">
						<div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div class="min-w-0">
								<h3 class="text-base font-medium text-warm">{{.Title}}</h3>
								<p class="text-xs text-warm-faint mt-1">{{.TargetRepo}} · {{.TargetRef}}</p>
							</div>
							<div class="flex gap-2 flex-wrap">
								<span class="text-[10px] px-2 py-1 rounded-full border border-edge text-warm-faint bg-void/30 whitespace-nowrap">{{.ApprovalState}}</span>
								<span class="text-[10px] px-2 py-1 rounded-full border border-edge text-warm-faint bg-void/30 whitespace-nowrap">residual: {{.ResidualState}}</span>
								<span class="text-[10px] px-2 py-1 rounded-full border border-brand/30 text-brand bg-brand/10 whitespace-nowrap">display only</span>
							</div>
						</div>
						<p class="text-xs text-warm-muted leading-relaxed">{{.Summary}}</p>
						<dl class="grid gap-x-3 gap-y-2 text-xs md:grid-cols-[9rem_1fr]">
							<dt class="text-warm-faint">required_head</dt><dd class="text-warm-muted break-all">{{if .RequiredHead}}{{.RequiredHead}}{{else}}missing{{end}}</dd>
							<dt class="text-warm-faint">approved_head</dt><dd class="text-warm-muted break-all">{{if .ApprovedHead}}{{.ApprovedHead}}{{else}}missing{{end}}</dd>
							<dt class="text-warm-faint">approval_source</dt><dd class="text-warm-muted break-all">{{if .ApprovalSourceURL}}<a href="{{.ApprovalSourceURL}}" rel="noopener" class="text-brand hover:text-brand/80">{{.ApprovalSourceURL}}</a>{{else}}missing{{end}}</dd>
							<dt class="text-warm-faint">approval_actor</dt><dd class="text-warm-muted">{{if .ApprovalActor}}{{.ApprovalActor}}{{else}}missing{{end}}</dd>
							<dt class="text-warm-faint">limitation</dt><dd class="text-warm-muted">{{.Limitation}}</dd>
						</dl>
					</article>
				{{end}}
			</div>
		</section>

		<section class="border border-edge bg-surface rounded-lg overflow-hidden" data-external-committee-queue="read-only">
			<header class="px-4 py-3 border-b border-edge">
				<h2 class="text-sm font-medium text-warm">External Committee decision queue</h2>
				<p class="text-xs text-warm-faint mt-1">Issue-sourced human-required decisions and protected-action blockers. Queue rows are display-only and cannot approve, deny, merge, label, comment, or close anything.</p>
			</header>
			<div class="divide-y divide-edge">
				{{range .DecisionQueue}}
					<article class="p-4 space-y-3" data-external-committee-queue-item="{{.ID}}" data-queue-state="{{.QueueState}}" data-evidence-state="{{.EvidenceState}}" data-display-only="{{.DisplayOnly}}">
						<div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div class="min-w-0">
								<h3 class="text-base font-medium text-warm">{{.Title}}</h3>
								<p class="text-xs text-warm-faint mt-1">{{.SourceIssue}} · {{.SourceRepo}}</p>
							</div>
							<div class="flex gap-2 flex-wrap">
								<span class="text-[10px] px-2 py-1 rounded-full border border-edge text-warm-faint bg-void/30 whitespace-nowrap">{{.QueueState}}</span>
								<span class="text-[10px] px-2 py-1 rounded-full border border-edge text-warm-faint bg-void/30 whitespace-nowrap">evidence: {{.EvidenceState}}</span>
								<span class="text-[10px] px-2 py-1 rounded-full border border-brand/30 text-brand bg-brand/10 whitespace-nowrap">display only</span>
							</div>
						</div>
						<dl class="grid gap-x-3 gap-y-2 text-xs md:grid-cols-[9rem_1fr]">
							<dt class="text-warm-faint">source</dt><dd class="text-warm-muted break-all">{{if .SourceURL}}<a href="{{.SourceURL}}" rel="noopener" class="text-brand hover:text-brand/80">{{.SourceURL}}</a>{{else}}missing{{end}}</dd>
							<dt class="text-warm-faint">required_actor</dt><dd class="text-warm-muted">{{.RequiredActor}}</dd>
							<dt class="text-warm-faint">decision_needed</dt><dd class="text-warm-muted">{{.DecisionNeeded}}</dd>
							<dt class="text-warm-faint">blocker</dt><dd class="text-warm-muted">{{.Blocker}}</dd>
							<dt class="text-warm-faint">no_write_limit</dt><dd class="text-warm-muted">{{.NoWriteLimit}}</dd>
						</dl>
					</article>
				{{end}}
			</div>
		</section>

		<section class="border border-edge bg-surface rounded-lg overflow-hidden" data-review-console="read-only">
			<header class="px-4 py-3 border-b border-edge">
				<h2 class="text-sm font-medium text-warm">Review items</h2>
				<p class="text-xs text-warm-faint mt-1">Each item is display-only and fails closed when evidence is missing, stale, pending, or carried as a residual.</p>
			</header>
			<div class="divide-y divide-edge">
				{{range .Items}}
					<article class="p-4 space-y-3" data-review-item="{{.ID}}" data-display-only="{{.DisplayOnly}}" data-evidence-state="{{.EvidenceState}}" data-residual-state="{{.ResidualState}}">
						<div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div class="min-w-0">
								<h3 class="text-base font-medium text-warm">{{.Title}}</h3>
								<p class="text-xs text-warm-faint mt-1">{{.DecisionKind}} · {{.GateScope}}</p>
							</div>
							<div class="flex gap-2 flex-wrap">
								<span class="text-[10px] px-2 py-1 rounded-full border border-edge text-warm-faint bg-void/30 whitespace-nowrap">{{.EvidenceState}}</span>
								<span class="text-[10px] px-2 py-1 rounded-full border border-edge text-warm-faint bg-void/30 whitespace-nowrap">residual: {{.ResidualState}}</span>
								<span class="text-[10px] px-2 py-1 rounded-full border border-brand/30 text-brand bg-brand/10 whitespace-nowrap">display only</span>
							</div>
						</div>
						<dl class="grid gap-x-3 gap-y-2 text-xs md:grid-cols-[9rem_1fr]">
							<dt class="text-warm-faint">source</dt><dd class="text-warm-muted break-all"><a href="{{.SourceURL}}" rel="noopener" class="text-brand hover:text-brand/80">{{.SourceURL}}</a></dd>
							<dt class="text-warm-faint">source_type</dt><dd class="text-warm-muted">{{.SourceType}}</dd>
							<dt class="text-warm-faint">source_repo</dt><dd class="text-warm-muted">{{.SourceRepo}}</dd>
							<dt class="text-warm-faint">exact_head</dt><dd class="text-warm-muted break-all">{{if .ExactHead}}{{.ExactHead}}{{else}}not applicable{{end}}</dd>
							<dt class="text-warm-faint">required_actor</dt><dd class="text-warm-muted">{{.RequiredActor}}</dd>
							<dt class="text-warm-faint">required_action</dt><dd class="text-warm-muted">{{.RequiredAction}}</dd>
							<dt class="text-warm-faint">limitation</dt><dd class="text-warm-muted">{{.Limitation}}</dd>
						</dl>
					</article>
				{{end}}
			</div>
		</section>

		<p class="text-[11px] text-warm-faint">Event 13 Level 0 read-only decision evidence surface. No GitHub write, protected action, runtime execution, EventGraph write, deploy, production go-live, autonomy increase, value allocation, docs#172 closure, Test 001 closure, live-reader proof, public-correction proof, or residual-risk closure.</p>
	</main>
</body>
</html>`))
