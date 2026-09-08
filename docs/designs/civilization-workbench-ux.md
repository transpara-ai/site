# Civilization Workbench interaction design

Route: Designed. The behavior choice is a work index with one selected detail,
instead of a page that expands every item and always exposes the intake form.

Outcome: operators can find what needs their attention, understand progress,
confirm an exact brief, recover blocked work, and inspect the prepared result.

The index groups human attention first, then active work, prepared results,
review-ready work, and completed work. Within groups, recent updates come first.
Selection has a normal URL and survives refresh; a missing explicit selection
never silently changes to a different work item. Mobile uses a native disclosure
for the index. New work opens a compact composer; model overrides are optional.

Board actions update only the board. The unrelated intake draft remains mounted.
Progress updates automatically every three seconds, including during focused
interaction, pending submissions and errors. There is no manual refresh control.
Poll responses started before a newer action are discarded; unavailable responses
retain the last working board and reconnect automatically. Focus, cursor position,
pending submissions and edited assignment forms are preserved during updates.
Intervention drafts and evidence disclosure state survive board refreshes and
selection changes in memory. Status announcements occur only when selected work,
state, or owner changes. Native links/forms work without JavaScript.
HTMX history snapshots are disabled so task text is not saved in browser storage;
back/forward restoration fetches the selected work again. Appearance is the only
preference written to local storage.

The Team board rolls up concurrent workstreams with repository, requester and
responsible-person filters, workflow stages, configured host, errors and pending
human steps. Requesters resolve through Site's existing user directory; missing
identity stays explicit. Standard sun, moon and monitor buttons choose light,
dark and system appearance.

Human responsibility is a durable Hive assignment with an expected assignment
version. An operator chooses a known workbench participant or returns work to the
shared unassigned queue. Site supplies the authenticated assigning operator ID;
an assignment grants no new approval permission. Conflicting writes produce a
visible error instead of replacing the other operator's choice. The assignment
applies to the workstream's current and future human steps. Requesters and human
owners are separate fields. The default is unassigned until explicitly assigned.

Hive's reconciler already defaults to three parallel workstreams. Each work has
its own lock, isolated worktree and durable event history. The configured runtime
limit is not exposed in the work API, so the UI does not pretend to report free
worker slots. The runtime roster reuses the existing Hive operator projection and
refreshes independently every ten seconds. It reports observed agent names,
roles and models, with explicit freshness; that projection does not currently
associate its actors with individual work IDs. A work's stage is not a heartbeat.

The four stages are Brief, Implementation, Checks & review, and Result. Checks
and review share a stage because the caller independently verifies after provider
review. A prepared result does not imply publication or merge. Unknown states and
checks do not acquire a success label. Diff HTML is escaped and its preview is
bounded; the complete existing text artifact remains downloadable.

Scope: Site templates, scoped light/dark CSS, minimal HTMX enhancement, and Site
presentation handlers, consuming Hive's separate human-responsibility endpoint.
Existing authenticated routes, server-side credentials,
Hive execution, exact brief binding, publication and merge controls remain in place.
No production deployment or provider configuration is part of this change.

Validation: native `make verify`; focused selection, brief binding, rollup,
assignment identity/version and artifact tests; actual Site browser coverage for
two simultaneous sessions, remote updates, assignment conflicts, owner/requester
filters, desktop/mobile, keyboard use, theme icons, live reconnect, focused drafts,
retained errors and inline artifacts; Platform's persisted Civilization
journey for Site → Hive → signed evidence → prepared artifact. The Site browser
fixture uses deterministic API states; it does not qualify a live provider.

To repeat the browser check after `make build`, run the fixture in one terminal:

```sh
WORKBENCH_BROWSER_URL_FILE=/tmp/site-workbench-url go test ./graph -run '^TestWorkbenchBrowserFixture$' -count=1 -v
```

After the URL file is written, run in another terminal (with Playwright available):

```sh
WORKBENCH_BROWSER_URL_FILE=/tmp/site-workbench-url node scripts/test-civilization-workbench.cjs
```

`PLAYWRIGHT_MODULE` accepts an existing Playwright module path, and
`WORKBENCH_BROWSER_OUTPUT` accepts a screenshot directory. The script stops the
temporary fixture server. For the persisted journey, set Platform's
`CIVILIZATION_BROWSER_URL_FILE`, pass that same file here, and set
`WORKBENCH_BROWSER_KIND=persisted`.
