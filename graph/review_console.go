package graph

import (
	"html/template"
	"net/http"
)

type OpsReviewConsoleData struct {
	Title               string
	GeneratedAt         string
	AuthorizationSource string
	Boundary            []string
	Items               []OpsReviewItem
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
		GeneratedAt:         "static Event 13 fixture",
		AuthorizationSource: "docs v4.0 Event 13 AuthorityDecision, merged by docs#185",
		Boundary: []string{
			"display_only=true for every review item",
			"no approve, reject, merge, close, label, comment, deploy, RuntimeBroker, EventGraph write, or protected action path",
			"Site is a console, not the truth source",
			"Gate W remains open until a later docs evidence-decision PR accepts, rejects, or defers the Site implementation evidence",
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
				Limitation:     "Approval plus merge grants only one future bounded Site PR lifecycle; it does not close Gate W.",
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
				SourceURL:      "https://github.com/transpara-ai/civilization-operation/issues/26",
				SourceType:     "issue",
				SourceRepo:     "transpara-ai/civilization-operation",
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
