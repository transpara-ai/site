// Package profile defines the request-time Profile abstraction that lets
// the site host multiple visual identities under a single URL space.
//
// Phase 4 (first divergence): a Profile now carries enough data to
// produce visibly different HTML — brand name, logo path, accent
// color. Byte-identical-output is over; templates read these fields
// through nil-safe accessors so a missing Profile falls back to the
// default branding rather than panicking.
package profile

import "log/slog"

// NavItem is a single link in a profile-driven navigation bar. Path is
// the href value (starting with "/" for internal routes); Label is the
// visible link text. Each profile's HeaderNav / FooterNav is an ordered
// slice of these — order = render order.
type NavItem struct {
	Label string
	Path  string
}

// Profile is the declarative presentation bundle attached to a request.
//
// Slug is the resolution key (query param, subdomain, etc.) and
// uniquely identifies the profile. BrandName, LogoPath, and
// AccentColor drive the visible chrome. HeaderNav and FooterNav drive
// the profile-aware nav surfaces — the public Layout, simpleHeader
// (orphan pages), and appLayout (app chrome) all read HeaderNav; the
// public Layout footer reads FooterNav. Copy is a sparse string
// override map (key → value); templates call p.GetCopy(key, fallback)
// and the profile only overrides the specific keys it cares about.
// Name is retained for backward compatibility with earlier phases —
// new code should read BrandName.
type Profile struct {
	Slug        string // resolution key
	Name        string // display label (kept for backward compat)
	BrandName   string // what appears in headers, titles, meta
	LogoPath    string // path to logo asset (e.g. /static/logo-lovyou.svg)
	AccentColor string // hex color for CSS --accent property

	HeaderNav []NavItem         // items rendered in the top chrome nav
	FooterNav []NavItem         // items rendered in the public layout footer
	Copy      map[string]string // sparse string overrides keyed by call-site key
}

// DefaultSlug is the slug that resolves when no other resolver matches.
const DefaultSlug = "transpara-ai"

// legacyAliases maps deprecated profile slugs to their canonical post-rename
// form. Exists to catch external state (DB rows, Host headers, static config,
// bookmarked URLs) that still references the pre-2026-04-22 slug after the
// Prompt 2 sweep renamed the registry. Every resolution of a legacy slug
// emits a structured deprecation warning so operators can grep logs for the
// request source and migrate it. Drop this map once the warnings stop firing
// for a sustained window.
var legacyAliases = map[string]string{
	"lovyou-ai": "transpara-ai",
}

// registry is the authoritative set of known profiles. Keyed by Slug.
// Phase 4 is the point at which the entries intentionally diverge —
// different BrandName, different LogoPath, different AccentColor
// produce visibly different chrome per request.
var registry = map[string]*Profile{
	"transpara-ai": {
		Slug:        "transpara-ai",
		Name:        "lovyou.ai",
		BrandName:   "lovyou.ai",
		LogoPath:    "/static/logo-lovyou.svg",
		AccentColor: "#e8a0b8",
		HeaderNav: []NavItem{
			{Label: "Discover", Path: "/discover"},
			{Label: "Hive", Path: "/hive"},
			{Label: "Agents", Path: "/agents"},
			{Label: "Blog", Path: "/blog"},
		},
		FooterNav: []NavItem{
			{Label: "Discover", Path: "/discover"},
			{Label: "Hive", Path: "/hive"},
			{Label: "Agents", Path: "/agents"},
			{Label: "Market", Path: "/market"},
			{Label: "Knowledge", Path: "/knowledge"},
			{Label: "Activity", Path: "/activity"},
			{Label: "Search", Path: "/search"},
			{Label: "Blog", Path: "/blog"},
			{Label: "Reference", Path: "/reference"},
		},
	},
	"transpara": {
		Slug:        "transpara",
		Name:        "Transpara",
		BrandName:   "Transpara",
		LogoPath:    "/static/logo-transpara.svg",
		AccentColor: "#0ea5e9",
		// Transpara's nav intentionally omits hive, agents, market,
		// knowledge, activity, and reference — those routes surface
		// lovyou.ai-specific civilization-build content (the live
		// agent timeline, the agent persona catalogue, the task
		// marketplace, the knowledge-claim graph, the site-wide
		// activity stream, the grammar reference). Discover (public
		// spaces) and Blog (long-form content) are generic platform
		// surfaces that still make sense under a Transpara identity.
		HeaderNav: []NavItem{
			{Label: "Discover", Path: "/discover"},
			{Label: "Blog", Path: "/blog"},
		},
		FooterNav: []NavItem{
			{Label: "Discover", Path: "/discover"},
			{Label: "Blog", Path: "/blog"},
		},
		// Copy overrides target a small set of brand-bearing
		// sentences whose tone differs sharply from the default
		// profile's. Anything not in this map renders the
		// fallback string the call site supplies — same as today.
		// Keep this set small (~10-15 keys) until a real i18n
		// system absorbs it; this is a pattern proof, not a
		// translation infrastructure.
		Copy: map[string]string{
			"tagline":                 "Operations intelligence for the people on the floor.",
			"home.description":        "Autonomous agents read your plant data and surface what matters — before it costs you a shift.",
			"home.hero.title.lead":    "See what the plant",
			"home.hero.title.accent":  "is telling you.",
			"home.hero.subtitle":      "Surface what the plant is telling you, before it costs you a shift. The data was always there — now your team can act on it.",
			"home.subhero.body":       "Autonomous agents reason about your operation in real time. Watch how they connect signals across the plant.",
			"app.description":         "Your team's work on the plant — tasks, conversations, and agent activity in one place.",
			"hive.description":        "Live view of the agents working on the plant — real-time activity and coordination across the floor.",
			"welcome.description":     "Set up your first workspace and start asking questions of the plant data.",
			"notifications.description": "Recent activity across your plant workspaces.",
			"discover.empty":          "No shared workspaces yet.",
			"welcome.subtitle":        "You're in. Let's set up your first workspace — somewhere your team can ask questions of the plant data. Takes 30 seconds.",
			"apikeys.desc":            "Authenticate scripts and agents to query Transpara programmatically — the same data your team sees, accessible from your tooling.",
		},
	},
}

// Default returns the fallback profile used when no resolver matches.
func Default() *Profile {
	return registry[DefaultSlug]
}

// Lookup returns the profile for the given slug, or nil if unknown.
// Callers use nil as the signal to continue resolver-chain traversal.
//
// Resolution order:
//  1. Exact match in the registry (the fast, canonical path)
//  2. Legacy alias (emits a deprecation warning, returns the canonical profile)
//  3. nil (unchanged — caller's resolver chain falls through to DefaultResolver)
//
// Legacy-alias resolution returns the *canonical* Profile, so callers rendering
// Profile.Slug (e.g. data-profile HTML attribute via GetSlug) render the
// post-migration identifier — the requested slug string is not leaked into the
// returned Profile.
func Lookup(slug string) *Profile {
	if p, ok := registry[slug]; ok {
		return p
	}
	if canonical, ok := legacyAliases[slug]; ok {
		slog.Warn("deprecated profile slug resolved via legacy alias",
			"requested", slug,
			"canonical", canonical,
			"action", "migrate external references to canonical slug")
		return registry[canonical]
	}
	return nil
}

// All returns a slice of every registered profile. Order is not
// guaranteed — iteration helper for tests and future admin surfaces.
func All() []*Profile {
	out := make([]*Profile, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}

// GetSlug returns the profile slug, falling back to the default on nil.
// Templates that emit slug-bearing attributes (e.g. data-profile) must
// route through this accessor so a nil *Profile renders default-profile
// identity instead of nil-panicking.
func (p *Profile) GetSlug() string {
	if p == nil {
		return Default().Slug
	}
	return p.Slug
}

// GetBrandName returns the brand name for rendering, falling back to
// the default profile's brand name if the receiver is nil. Templates
// must call this instead of reading p.BrandName directly so that any
// code path which produces a nil *Profile renders default branding
// instead of nil-panicking.
func (p *Profile) GetBrandName() string {
	if p == nil {
		return Default().BrandName
	}
	return p.BrandName
}

// GetLogoPath returns the logo asset path, falling back to default on nil.
func (p *Profile) GetLogoPath() string {
	if p == nil {
		return Default().LogoPath
	}
	return p.LogoPath
}

// GetAccentColor returns the accent color hex, falling back to default on nil.
func (p *Profile) GetAccentColor() string {
	if p == nil {
		return Default().AccentColor
	}
	return p.AccentColor
}

// GetHeaderNav returns the profile's top-chrome nav items, falling
// back to the default profile's HeaderNav if the receiver is nil OR
// has an empty slice. An empty slice is treated the same as nil here
// because a profile with zero header links would render a blank nav
// area — almost certainly a misconfiguration, not an intentional
// choice. Templates iterate the returned slice directly.
func (p *Profile) GetHeaderNav() []NavItem {
	if p == nil || len(p.HeaderNav) == 0 {
		return Default().HeaderNav
	}
	return p.HeaderNav
}

// GetFooterNav returns the profile's footer nav items, with the same
// nil-and-empty fallback semantics as GetHeaderNav.
func (p *Profile) GetFooterNav() []NavItem {
	if p == nil || len(p.FooterNav) == 0 {
		return Default().FooterNav
	}
	return p.FooterNav
}

// GetCopy returns the override registered for key on this profile, or
// fallback if no override exists. The fallback parameter means the
// call site provides its own default text inline — templates never
// have to know which keys exist, and a profile only declares the keys
// it actually wants to override.
//
// Nil receiver and nil/empty Copy map both return the fallback,
// matching the rest of the accessor family's "missing → default
// rendering" discipline.
func (p *Profile) GetCopy(key, fallback string) string {
	if p == nil || p.Copy == nil {
		return fallback
	}
	if v, ok := p.Copy[key]; ok {
		return v
	}
	return fallback
}
