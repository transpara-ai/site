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

Board actions refresh only the board. The unrelated intake draft remains mounted.
Polling pauses during focused board interaction, pending submissions, and errors.
Intervention drafts and evidence disclosure state survive board refreshes and
selection changes in memory. Status announcements occur only when selected work,
state, or owner changes. Native links/forms work without JavaScript.
HTMX history snapshots are disabled so task text is not saved in browser storage;
back/forward restoration fetches the selected work again. Appearance is the only
preference written to local storage.

The four stages are Brief, Implementation, Checks & review, and Result. Checks
and review share a stage because the caller independently verifies after provider
review. A prepared result does not imply publication or merge. Unknown states and
checks do not acquire a success label. Diff HTML is escaped and its preview is
bounded; the complete existing text artifact remains downloadable.

Scope: Site templates, scoped light/dark CSS, minimal HTMX enhancement, and Site
presentation handlers. Existing authenticated routes, server-side credentials,
Hive execution, exact brief binding, publication and merge controls remain in place.
No production deployment or provider configuration is part of this change.

Validation: native `make verify`; focused selection, brief binding and artifact
tests; actual Site browser coverage for desktop/mobile, keyboard use, themes,
polling, drafts, failures and inline artifacts; Platform's persisted Civilization
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
