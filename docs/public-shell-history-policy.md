# Public Shell History Policy

The public shell is the active Transpara-AI Civilization command surface and
must not render legacy lovyou-era marketing identity, colors, or value slogans.

Historical material may remain in provenance-bearing locations:

- `content/**`
- `third_party/**`
- `docs/**`
- tests and generated review evidence

Those paths preserve source history and reference material. They are not global
navigation, homepage, auth, or product/app chrome.

`make verify` runs `scripts/verify-public-shell-clean.sh` against active shell
and app paths. Add new allowlisted history paths deliberately; do not weaken the
active-shell scan to make a product-surface regression pass.
