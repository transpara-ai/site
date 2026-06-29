# Site 193 Operation Public-Proof Reference Evidence

Captured: 2026-06-29

Scope: local render evidence for the display-only `/ops/public-proof` Site
surface after adding the Operation-approved public-proof packet reference.

This evidence confirms that Site displays the Operation packet as an
`operation-reference` / `projection-only` record, shows `reference_generated_at`
and `reference_fresh_until`, and keeps public-reader proof and
public-correction proof `unavailable`.

## Files

| File | Viewport | Route | Purpose |
| --- | --- | --- | --- |
| [public-proof-desktop-full.png](public-proof-desktop-full.png) | 1440 x 1200, full page | `/ops/public-proof?profile=transpara` | Confirms desktop rendering shows the Operation packet reference, freshness metadata, display-only boundary, required labels, and unavailable proof rows. |
| [public-proof-mobile-full.png](public-proof-mobile-full.png) | 390 x 844, full page | `/ops/public-proof?profile=transpara` | Confirms mobile rendering keeps the packet reference, freshness metadata, and unavailable proof rows readable without overlap. |

## Capture Method

The Site binary was run locally with `PORT=18093`. Screenshots were captured with
Playwright CLI after waiting for `[data-public-proof="display-only"]`.

## Operation Source Verification

Verified on 2026-06-29 against `transpara-ai/operation@main`:

| Evidence | Value |
| --- | --- |
| `origin/main` commit | `7ab3929ff88d0b75c48d53a80e92db74ec523482` |
| Rendered evidence href | `https://github.com/transpara-ai/operation/blob/7ab3929ff88d0b75c48d53a80e92db74ec523482/docs/operations/public-proof-evidence/public-proof-reference-2026-06-29.md` |
| GitHub contents path | `docs/operations/public-proof-evidence/public-proof-reference-2026-06-29.md` |
| GitHub contents blob SHA | `f7896edf0fa8847aaa8411730db9d493418ff7d9` |
| Record id | `OPERATION-PUBLIC-PROOF-REFERENCE-2026-06-29` |
| Status | `reference/unavailable-proof` |
| Generated at | `2026-06-29T08:10:49Z` |
| Proof class | `operation-approved-public-proof-reference` |
| Fresh until | `2026-07-06T08:10:49Z` |
| Stale after days | `7` |

Validation commands run before capture:

```text
git diff --check
go test -count=1 ./graph -run 'TestHandleOpsPublicProofRendersDisplayOnlyEvidenceLedger|TestBuildOpsPublicProofDataMarksOperationPacketStaleAfterFreshUntil|TestBuildOpsPublicProofDataKeepsReaderAndCorrectionProofUnavailable'
make verify
```

## Boundary

This is local Site render evidence only. It does not prove deploy, live public
reader access, public-correction proof, runtime execution, EventGraph writes,
Hive wake, Test 001 GREEN or closure, `operation#26` closure, `operation#45`
closure, residual-risk closure, value allocation, production go-live, or
autonomy increase.
