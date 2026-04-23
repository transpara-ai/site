# Artifact 09 Path Correction Prompt

**Version:** 0.1.0 · **Date:** 2026-04-21
**Author:** Claude Opus 4.7
**Owner:** Michael Saucier
**Status:** Ready to execute — copy into Claude Code session (Session 2 of 3) AFTER Session 1's PR is merged
**Purpose:** Update Artifact 09 (Phase 3 prompt) to reflect the file relocation in PR #21. Findings docs moved from repo root into `docs/site-profile-redesign/`. Artifact 09's Precondition and Companion sections still reference repo-root paths; this prompt fixes them.
**Companion:** `09-prompt-3-phase-3-profile-context.md` (v0.1.0) — the artifact being patched.

---

### Revision History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.0 | 2026-04-21 | Initial path correction prompt. Scope: update Artifact 09's Precondition bullets and Companion line to reflect the docs/site-profile-redesign/ relocation done in PR #21. Bump Artifact 09 patch version to 0.1.1. Add a Revision History row. No other content changes. |

---

## Context (for the operator — not for Claude Code)

Artifact 09 was drafted before PR #21 ran. Two sections refer to file locations that have since changed:

1. **Precondition section** says *"Both `phase-1-token-refactor-findings-v0.1.0.md` and `phase-2-tailwind-build-step-findings-v0.1.0.md` should be readable at repo root."* They're not at repo root anymore. PR #21 relocated them to `docs/site-profile-redesign/`.

2. **Companion line** in frontmatter references the findings docs without paths, implying repo-root. Should now read the subdirectory path.

Both are single-file patches to Artifact 09. No retro, no investigation, no content change beyond the paths and a patch-version bump.

---

## How to launch

1. **Session 1 (doc hygiene) PR must be merged first.** Confirm via `gh pr list --state merged --limit 5` — the Session 1 PR should appear in recent merges.
2. **Launch Claude Code from the site repo root:**
   ```
   cd /Transpara/transpara-ai/data/repos/site/
   git checkout main && git pull --ff-only
   git status
   claude
   ```
3. **Paste the PROMPT block below** as your first message.
4. Review the PR it opens. Merge.

---

## PROMPT — copy everything between `─── BEGIN ───` and `─── END ───` markers

```
─── BEGIN PROMPT — ARTIFACT 09 PATH CORRECTION ───

ROLE
You are patching Artifact 09 (the Phase 3 prompt) in the
site repo to reflect the file relocation done in
PR #21. Findings docs moved from repo root into
docs/site-profile-redesign/. Artifact 09's Precondition and
Companion sections still reference repo-root paths.

GOAL
Update Artifact 09 so its path references match the current
state of the repo. Bump its version from 0.1.0 to 0.1.1. Add
a Revision History entry. No other content changes.

CONSTRAINTS (non-negotiable)
1. ONE FILE TOUCHED: Artifact 09 itself. Nothing else.
2. VERSION BUMP: 0.1.0 → 0.1.1 (patch bump — this is a
   correction, not a scope change).
3. REVISION HISTORY: add one row documenting the path update.
4. NO CONTENT CHANGES beyond:
   - Frontmatter Version field
   - Frontmatter Companion line (path corrections only)
   - Revision History table (one new row)
   - Precondition section (path corrections only)
   - Any other location in the prompt body that references
     the findings docs by path (search globally; see Pass 1)
5. NO MAIN-BRANCH COMMITS. Feature branch:
   docs/artifact-09-path-correction
6. CO-AUTHORED-BY TAG:
   Co-Authored-By: transpara-ai <transpara-ai@transpara.com>
7. PR, DO NOT MERGE.

WORK PLAN

Two passes.

PASS 1 — AUDIT (read-only)

1. Locate Artifact 09. Likely path:
   docs/site-profile-redesign/09-prompt-3-phase-3-profile-context.md
   If not there, search:
   find . -name '09-prompt-3-phase-3-profile-context*' -type f
   Record the exact path.

2. Confirm findings doc locations on current main:
   find docs/site-profile-redesign/ -name 'phase-1-token-refactor-findings*'
   find docs/site-profile-redesign/ -name 'phase-2-tailwind-build-step-findings*'
   Record each exact path (relative to repo root).

3. In Artifact 09, find every reference to the findings doc
   filenames. Search:
   grep -n 'phase-1-token-refactor-findings\|phase-2-tailwind-build-step-findings' <artifact-09-path>
   Record every line number and the surrounding context for
   each match.

4. In Artifact 09, find every reference to "repo root" that
   refers to the findings docs. Search:
   grep -ni 'repo root' <artifact-09-path>
   Not every "repo root" reference is about findings (some
   refer to where the Phase 3 findings doc will be WRITTEN,
   which is separate). For each match, record whether it
   refers to existing findings (needs update) or the future
   output of Phase 3 (does NOT need update — the Phase 3
   findings doc output path is its own decision).

5. Read the current frontmatter Companion line VERBATIM.
   Record which findings doc references appear there without
   a `docs/site-profile-redesign/` prefix.

6. Write the audit to /tmp/artifact-09-audit.md (scratch, not
   committed). Include:
   - Artifact 09 exact path
   - Current findings doc paths (from find command)
   - Every line number in Artifact 09 that needs updating
   - The full current Companion line text
   - The full current Precondition section text (lines
     mentioning findings)
   - Classification of "repo root" matches: finding-refs vs
     Phase-3-output-refs

Do NOT proceed to Pass 2 until the audit is complete and you
know exactly which lines change.

PASS 2 — EDIT + COMMIT + PR

2a. BRANCH

   git checkout -b docs/artifact-09-path-correction

2b. EDIT 1: Frontmatter Version bump

   Change the Version field from:
     **Version:** 0.1.0 · **Date:** 2026-04-21
   To:
     **Version:** 0.1.1 · **Date:** 2026-04-21

   (Date stays 2026-04-21 — this patch is same-day. If the
   current date when you run this is different, use today's
   actual date.)

2c. EDIT 2: Frontmatter Companion line

   Any reference in the Companion line to:
     phase-1-token-refactor-findings-v0.1.0.md
     phase-2-tailwind-build-step-findings-v0.1.0.md
   becomes:
     docs/site-profile-redesign/phase-1-token-refactor-findings-v0.1.0.md
     docs/site-profile-redesign/phase-2-tailwind-build-step-findings-v0.1.0.md

   (Prefix the subdirectory path.)

   Other entries in the Companion line that already have
   correct paths stay unchanged.

2d. EDIT 3: Revision History table

   Add a new row at the TOP of the table data rows (below
   the header row). New row:

   | 0.1.1 | 2026-04-21 | Path correction: findings docs relocated to `docs/site-profile-redesign/` in PR #21. Updated Precondition bullets and Companion references. No scope change. |

   The existing 0.1.0 row stays; it just moves down one
   position in the table order.

2e. EDIT 4: Precondition section

   Find the Precondition bullets that mention the findings
   docs "at repo root" (identified in Pass 1 step 4).

   Replace each mention of "at repo root" with
   "in docs/site-profile-redesign/" for the specific bullets
   that refer to the findings docs.

   Do NOT change any bullet that refers to:
   - main being clean at HEAD
   - git status being clean
   - `go test ./...` passing on main
   - `make css` producing site.css on main
   - Any other precondition unrelated to findings doc paths

2f. EDIT 5: Any other in-body references

   For each line number recorded in Pass 1 step 3 that has
   NOT already been updated by edits 2c–2e, update it to use
   the correct `docs/site-profile-redesign/` prefix.

   Do NOT change lines that refer to the Phase 3 findings
   doc OUTPUT path — that file does not exist yet and its
   output location is a separate decision (it will be
   written to repo root during Phase 3 execution, per the
   prompt's DELIVERABLE section). Leave those lines alone.

2g. VERIFY

   git diff main -- <artifact-09-path>

   The diff must show:
   - Version line changed (0.1.0 → 0.1.1)
   - Companion line updated (paths prefixed)
   - Revision History has one new row above the existing row
   - Precondition bullets about findings docs updated
   - Any other references identified in Pass 1 updated

   The diff must NOT show:
   - Changes to any other file
   - Changes to the DELIVERABLE section's output path
     specification
   - Wording changes beyond path corrections and the new
     Revision History row

   If the diff shows anything unexpected, STOP and investigate
   before committing.

2h. COMMIT

   git add <artifact-09-path>
   git commit -m "docs: correct artifact 09 path references to docs/site-profile-redesign/" \
              -m "Findings docs were relocated in PR #21. This patch updates Artifact 09's Precondition, Companion, and any in-body references. Bumps Artifact 09 version 0.1.0 → 0.1.1." \
              -m "Co-Authored-By: transpara-ai <transpara-ai@transpara.com>"

2i. PUSH AND PR

   git push -u origin docs/artifact-09-path-correction

   gh pr create --base main \
     --title "docs: correct artifact 09 path references" \
     --body "Patches Artifact 09 (Phase 3 prompt) to reflect the findings-doc relocation done in PR #21.

Changes:
- Version bumped 0.1.0 → 0.1.1
- Companion line paths prefixed with \`docs/site-profile-redesign/\`
- Precondition bullets updated to reference the new location
- Revision History updated with the 0.1.1 row
- No scope change; no content change beyond path corrections

One file touched, one commit."

   STOP. Do not merge.

DELIVERABLE

A single PR on origin/main:
- 1 commit
- Only Artifact 09 modified
- Co-author trailer on the commit
- PR open, not merged

If any verification check in 2g fails, STOP and report.

APPROACH HINTS

1. The "at repo root" → "in docs/site-profile-redesign/"
   substitution is only for bullets about findings docs.
   The Phase 3 prompt's OWN deliverable — where Phase 3's
   findings doc gets written — stays at repo root per the
   DELIVERABLE section. Don't conflate these.
2. If you find a reference to the findings docs that you're
   not sure should be updated, err on the side of leaving
   it alone and note it in the PR body as "reviewed but
   not changed — Michael to confirm."
3. The Revision History row's description field should
   clearly name PR #21 as the cause. Future readers need
   to understand why the path changed.

FINAL CHECK BEFORE YOU START

- [ ] Working directory is site repo root
- [ ] Current branch is main, clean status
- [ ] Session 1 (doc hygiene) PR is merged
- [ ] You understand: one file, one commit, path corrections
      + version bump + revision history row
- [ ] You will produce /tmp/artifact-09-audit.md in Pass 1
- [ ] You will open a PR and STOP — not merge

Begin.

─── END PROMPT — ARTIFACT 09 PATH CORRECTION ───
```

---

## What happens after this runs

1. CC audits Artifact 09, creates a branch, edits one file, commits, pushes, opens PR.
2. You review. Check: one file changed, version bumped to 0.1.1, Revision History updated, no scope drift.
3. Merge.
4. Proceed to Session 3: actually execute Phase 3.

---

## Risk notes

**Risk 1 — More references than expected.** I drafted Artifact 09 in one sitting; I may have referenced the findings docs in more than just the Companion and Precondition sections. Pass 1's step 3 (`grep -n` for the findings filenames) surfaces every reference. If the grep returns more than 3–4 hits, that's fine — CC updates them all under Edit 5.

**Risk 2 — DELIVERABLE output path confusion.** Artifact 09's DELIVERABLE section says "Create `$REPO_ROOT/phase-3-profile-context-findings-v0.1.0.md`" — that's Phase 3's own future findings doc, which legitimately does go to repo root at creation time. If you want Phase 3's findings to land in `docs/site-profile-redesign/` directly instead (more consistent with PR #21's pattern), that's a SEPARATE decision and a SEPARATE patch — do not bundle it here. This prompt is strictly about correcting references to the *existing* Phase 1 and Phase 2 findings.

**Risk 3 — Date field.** The prompt says "use today's actual date" for the version bump. If CC runs this more than a day after drafting, its "today" may differ. That's fine — what matters is that the 0.1.1 row and the Version line both carry the date the patch was actually written.
