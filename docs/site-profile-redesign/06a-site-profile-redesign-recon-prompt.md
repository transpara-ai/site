# Prompt 0 — Site Profile Redesign Reconnaissance

**Version:** 0.1.0 · **Date:** 2026-04-20
**Author:** Claude Opus 4.7
**Owner:** Michael Saucier
**Status:** Ready to execute — copy into Claude Code session
**Versioning:** Versioned as part of the site-profile-redesign set (01–05). Major for structural changes to the recon scope; minor for additional recon targets; patch for corrections and clarifications.
**Companion:** `01-site-map-discovery.md`, `02-display-profile-system.md`, `03-transpara-profile-design.md`, `04-transpara-profile-wireframes.md`, `05-transpara-home-prototype.html`

---

### Revision History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.0 | 2026-04-20 | Initial recon prompt covering three repos (site, work, summary), four recon target areas, findings-doc deliverable, five-line summary format. Read-only by construction. |

---

## How to launch

1. **Attach Artifacts 01–05** to your Claude Code session (the five design documents). The recon references them; Claude Code cannot see them otherwise.
2. **Enable skills/plugins:**
   - `frontend-design` — for context when analyzing the existing frontend.
   - `hive-lifecycle` — if recon touches a running hive state (it shouldn't, but be ready).
3. **Launch from the parent directory** that has `site`, `work`, and `summary` as siblings. `$REPOS_ROOT` below is a placeholder for that path.
4. **Paste the PROMPT block below** into the session.

---

## PROMPT — copy everything between the `─── BEGIN ───` and `─── END ───` markers below

```
─── BEGIN PROMPT 0 — SITE PROFILE REDESIGN RECON ───

ROLE
You are performing a read-only reconnaissance pass against three sibling
repositories to verify a design before implementation begins. This is
Prompt 0 in the Transpara design-recon-revise-implement workflow. Your
output will be used to produce v0.3.0 corrections to design artifacts
01–05, which are attached to this session.

CONTEXT
I've produced five design artifacts for refactoring the site at
http://nucbuntu/ into a profile-driven system that supports multiple
visual identities (lovyou-ai today, Transpara next, others later). The
artifacts assume a specific stack, theming approach, and /hive wiring
strategy. Before I commit to the 8-step migration plan in Artifact 02 §6,
I need ground truth: what does the actual codebase look like?

The three repos you're investigating are siblings under the current
working directory:

  $REPOS_ROOT/
    ├── site/        ← PRIMARY refactor target
    ├── work/        ← /hive data source (telemetry API)
    └── summary/     ← existing telemetry dashboard

Your job is to surface surprises. Every major agent we've built (SysMon,
Allocator, CTO, Spawner) required v1.0.0 → v1.1.0+ corrections after
recon revealed things the design didn't anticipate — phantom event types,
wrong namespaces, missing fields, immutable runtime fields. This recon
is how we pay that debt up front instead of mid-implementation.

CONSTRAINTS (non-negotiable)
1. READ-ONLY. Do not modify any file. Do not run `git commit`. Do not
   create branches. Do not run `go build`, `npm install`, or anything
   that changes disk state. `ls`, `cat`, `find`, `grep`, `git log`,
   `git status`, `git diff --stat` are all fine.
2. SELF-CONTAINED. Do not `cd` out of the parent directory. All three
   repos are accessible from here.
3. FINDINGS-FIRST. If you encounter ambiguity, write questions in the
   findings document — do NOT make assumptions and do NOT extrapolate
   to code recommendations beyond what the recon supports.
4. NO IMPLEMENTATION. This prompt does not authorize changes. Even if
   you see an obvious fix, record it as a finding.

RECON TARGETS

─── A. site (the refactor target) ──────────────────────────

A.1 Stack identification
- Read `go.mod`, `package.json`, `Makefile`, `Dockerfile`, any build
  config at the repo root. Report: language(s), framework(s), template
  engine, CSS strategy (Tailwind / vanilla / CSS-in-JS / custom
  properties), JS strategy (HTMX / Alpine / vanilla / React).
- Identify the process boundary: is this one Go binary serving HTML
  templates, or a Go API + separate frontend build, or something else?

A.2 Theming surface
- Grep the repo for hardcoded colors in these forms:
    #[0-9a-fA-F]{3,8}   (hex)
    rgb(/rgba(          (functional)
    hsl(/hsla(          (functional)
- Grep for hardcoded font names (e.g. 'Inter', 'serif', 'Source Serif'),
  font-family declarations, @import url(...fonts...).
- Grep for CSS custom properties already in use: `var(--` and
  `:root` declarations. If CSS variables exist already, our token-
  extraction step is shorter than Artifact 02 §6 assumes — good news.
- Count the files containing hardcoded values. Report the top 10 files
  by hardcoded-color count. This tells us where the refactor pain lives.

A.3 Shell + layout structure
- Find the layout/template files. Common patterns:
    Go html/template: `templates/layout.html`, `templates/partials/*`
    Go templ:         `*.templ` files
    Next.js:          `app/layout.tsx` or `pages/_app.tsx`
    Astro:            `src/layouts/*.astro`
- Identify: where does the header live? Where does the footer live?
  Is there already a layout abstraction, or is chrome duplicated per
  page?
- How does the site render pages currently? Does each route have its
  own handler, or is there a single template-driven renderer?

A.4 Route map (verify Artifact 01)
- List every route handler registered in the site (grep for `HandleFunc`,
  `Handle`, `r.Get`, `app.get`, route tables — whatever the framework
  uses).
- Cross-check against Artifact 01 §1–§7. Report:
    - Routes in code but NOT in Artifact 01 (missed during browser recon)
    - Routes in Artifact 01 but NOT in code (stale references)
    - Route parameters we didn't document

A.5 /hive route — current implementation
- Locate the `/hive` handler and template.
- What does it render today? (The design says "Phase Timeline editorial
  page" under lovyou-ai profile.)
- Is there any existing proxy / iframe / API-call logic?
- What data does it consume?

A.6 Deployment + config
- How is this deployed to nucbuntu? systemd unit? Docker?
- What environment variables does it read at startup?
- Is there a config file? Where?

─── B. work (the /hive data source) ───────────────────────

B.1 Telemetry API surface
- List every HTTP endpoint exposed by the work-server.
- Identify the telemetry-specific endpoints specifically — the ones that
  back the dashboard at http://nucbuntu:8080/telemetry.
- For each telemetry endpoint, report: path, method, response shape
  (in broad strokes — no full schema dumps), auth requirement.

B.2 Auth model
- How does the telemetry API authenticate requests?
  - API key header? Bearer token? Both?
  - Where is the key validated?
  - Is there a dev key (`WORK_API_KEY=dev` is mentioned in prior docs)?
- Critical for Artifact 02 §7 Option 1 (reverse proxy): can the site
  server hold the API key server-side and front telemetry with its own
  session?

B.3 CORS + port
- Does the work-server emit CORS headers? For which origins?
- Is port 8080 fixed in code or env-configurable?
- Can the site server reach work-server via loopback (127.0.0.1:8080)
  on nucbuntu?

─── C. summary (the existing dashboard) ────────────────────

C.1 Dashboard artifact
- Locate the HTML file(s) that render the Transpara-AI Mission Control
  dashboard.
- Report: single file or multi-file? What JS libraries does it load?
  What assets (CSS/JS/images) does it reference?
- Are asset paths absolute (`/telemetry-assets/foo.js`) or relative
  (`./foo.js`)? This determines how much rewriting Artifact 02 §7
  Option 1 requires.

C.2 Embed-friendliness
- Does the dashboard have its own <html>, <head>, <body> chrome, or is
  it already a fragment?
- Is there any existing `?embed=1` mode, or any conditional chrome-
  hiding logic? (Artifact 02 §7 Option 2 depends on this.)
- Does it assume a specific parent origin, or does it work standalone?

C.3 Polling cadence + data dependencies
- How often does the dashboard poll the telemetry API? (Look for
  setInterval, fetch loops, WebSocket subscriptions.)
- Does it call any endpoint we didn't catch in Recon B.1?

─── D. Cross-cutting ─────────────────────────────────────────────────

D.1 Existing design-system traces
- Is there a shared CSS package, design-tokens file, or component
  library used across the three repos? Report any path that looks like
  a shared asset source.
- Any reference to a "profile", "theme", "brand", or "skin" concept
  already in the codebase? (Unlikely, but if yes, our design might be
  duplicating effort.)

D.2 Git branch posture
- For each repo: report current branch, `git status` summary (clean /
  dirty), last commit hash + subject + date. This establishes a recon
  timestamp.

D.3 CI/CD
- Any GitHub Actions / workflows across the three repos that deploy
  to nucbuntu? Report the trigger events and what they do at a high
  level.

D.4 Blockers
- Anything that makes the 8-step migration plan (Artifact 02 §6)
  impossible or substantially harder than assumed? Examples:
    - Theming is coupled to HTML structure (can't just swap tokens)
    - Routes are statically generated and can't accept a profile flag
    - The work-server is a separate deploy boundary that makes same-
      origin proxy impractical
    - `/hive` is generated at build time, not served dynamically
- Flag ruthlessly. It's cheaper to find blockers now.

DELIVERABLE

Create a markdown file at
`$REPOS_ROOT/site-profile-redesign-recon-findings-v0.1.0.md` with the
following structure. Do not skip sections; if a section has no findings,
say so explicitly.

  # Site Profile Redesign — Recon Findings

  **Version:** 0.1.0 · **Date:** <today>
  **Author:** Claude (via Claude Code)
  **Owner:** Michael Saucier
  **Status:** Recon complete — design correction pending
  **Companion:** site-profile-redesign-recon-prompt-v0.1.0.md

  ## A. site
  ### A.1 Stack
  ### A.2 Theming surface
  ### A.3 Shell + layout structure
  ### A.4 Route map — deltas vs Artifact 01
  ### A.5 /hive current implementation
  ### A.6 Deployment + config

  ## B. work
  ### B.1 Telemetry API surface
  ### B.2 Auth model
  ### B.3 CORS + port

  ## C. summary
  ### C.1 Dashboard artifact
  ### C.2 Embed-friendliness
  ### C.3 Polling cadence + data dependencies

  ## D. Cross-cutting
  ### D.1 Existing design-system traces
  ### D.2 Git branch posture
  ### D.3 CI/CD
  ### D.4 Blockers

  ## E. Recommended corrections to design artifacts

  Flat list of corrections needed in Artifacts 01–05. For each item:
    - Artifact + section reference (e.g. "Artifact 02 §7")
    - The design's current claim
    - What the codebase actually does
    - Suggested correction (describe the change, do not rewrite the
      artifact)

  ## F. Open questions for Michael

  Things where recon revealed ambiguity that only Michael can resolve.
  One question per line.

  ---

  ## Five-line summary

  Line 1: Stack (language · framework · theming strategy)
  Line 2: Biggest surprise vs design
  Line 3: /hive wiring — which of Artifact 02 §7 options 1/2/3 is
          actually easiest given findings
  Line 4: Blocker count + severity (or "no blockers")
  Line 5: Estimated correction scope (minor patch · 0.3.0 minor ·
          0.3.0 major restructure)

APPROACH HINTS
1. Start with `find $REPOS_ROOT -maxdepth 2 -type d` to confirm the three
   repos are where expected.
2. Read top-level build config before diving into source files. The
   stack determines where everything else lives.
3. For the theming grep, use `grep -rIn --include='*.css' --include='*.html'
   --include='*.templ' --include='*.go' --include='*.tsx' --include='*.jsx'`.
   Count by file, not by line — one busy file is noisier than 100
   single-hex files.
4. When reading templates, read the LAYOUT first and the CONTENT second.
   The layout is where the refactor happens.
5. For the /hive recon in A.5 and the dashboard recon in C.*, read the
   full file. These two are small enough and load-bearing for our
   strategy.
6. Timebox yourself: this recon should take under 90 minutes of
   investigation. If something requires deeper analysis, flag it in F
   rather than chasing it.

FINAL CHECK BEFORE YOU START
- [ ] Artifacts 01–05 are attached to this session
- [ ] Working directory is $REPOS_ROOT (has the three sibling repos)
- [ ] You will not modify any file
- [ ] You will produce the findings doc at the path specified
- [ ] You will end with the five-line summary

Begin recon.

─── END PROMPT 0 — SITE PROFILE REDESIGN RECON ───
```

---

## What happens after this runs

1. Claude Code produces `site-profile-redesign-recon-findings-v0.1.0.md` at `$REPOS_ROOT`.
2. You bring that findings doc back into the Claude.ai design thread (this one).
3. I ingest the findings, emit v0.3.0 of Artifacts 02, 03, 04 (patches or minor bumps depending on severity), and if warranted, Artifact 05 (the prototype may need updates too).
4. Then — and only then — we draft **Prompt 1: Token extraction + shell refactor** for Claude Code to begin Phase 1 of the migration plan.

No implementation happens until corrections land. This is the same discipline that saved SysMon and Allocator from shipping against bad assumptions.

---

## Why this recon is worth its weight

The biggest risk in our current design is Artifact 02 §7 (the three `/hive` wiring options). I picked reverse-proxy-plus-shell-injection as v1 recommended, but that recommendation assumes:

- The work-server emits CORS or is reachable same-origin from the site server
- The telemetry dashboard assets can be rewritten to absolute URLs under `/telemetry-assets/*`
- The telemetry dashboard can be stripped of its own HTML chrome cleanly

If any of those assumptions is wrong, the recommendation flips to the iframe approach (Option 2) or — worst case — forces the API-only rebuild (Option 3, currently listed as roadmap). The recon settles this before you burn days on the wrong branch.

Secondary risk: the theming-surface grep. If the lovyou-ai site already uses CSS custom properties extensively, Phase 1 of the migration plan is two days, not five. If it has 400+ hardcoded hex values spread across Go templates, Phase 1 is more like a week. Either way, knowing is cheaper than guessing.
