// Package graph implements the unified product backend — spaces, nodes, and
// grammar operations backed by Postgres. Replaces the separate work/social/market
// packages with one data model where every action is a grammar operation.
package graph

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/lib/pq"
)

// ────────────────────────────────────────────────────────────────────
// Domain types
// ────────────────────────────────────────────────────────────────────

// Node states.
const (
	StateOpen    = "open"
	StateActive  = "active"
	StateReview  = "review"
	StateBlocked = "blocked" // incomplete children or unresolved dependencies
	StateDone    = "done"
	StateClosed  = "closed"
)

// Task priorities.
const (
	PriorityUrgent = "urgent"
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Node kinds.
const (
	KindTask         = "task"
	KindPost         = "post"
	KindThread       = "thread"
	KindComment      = "comment"
	KindConversation = "conversation"
	KindClaim        = "claim"
	KindProposal     = "proposal"
	KindProject      = "project"
	KindGoal         = "goal"
	KindRole         = "role"
	KindTeam         = "team"
	KindPolicy       = "policy"
	KindDocument     = "document"
	KindQuestion     = "question"
	KindCouncil      = "council"
)

// Claim epistemic states.
const (
	ClaimClaimed    = "claimed"
	ClaimChallenged = "challenged"
	ClaimVerified   = "verified"
	ClaimRetracted  = "retracted"
)

// Proposal states.
const (
	ProposalOpen   = "open"
	ProposalPassed = "done"    // reuse "done" for passed proposals
	ProposalFailed = "closed"  // reuse "closed" for failed/rejected
)

// Space kinds.
const (
	SpaceProject   = "project"
	SpaceCommunity = "community"
	SpaceTeam      = "team"
)

// Space visibility.
const (
	VisibilityPrivate = "private"
	VisibilityPublic  = "public"
)

// Node membership ops.
const (
	OpJoinTeam  = "join_team"
	OpLeaveTeam = "leave_team"
)

// Governance ops.
const (
	OpDelegate   = "delegate"
	OpUndelegate = "undelegate"
)

// Voting body scopes for proposals.
const (
	VotingBodyAll     = "all"     // all space members
	VotingBodyCouncil = "council" // council node members
	VotingBodyTeam    = "team"    // team node members
)

// Space is a container — project, community, or team.
type Space struct {
	ID                 string     `json:"id"`
	Slug               string     `json:"slug"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	OwnerID            string     `json:"owner_id"`
	Kind               string     `json:"kind"`
	Visibility         string     `json:"visibility"`
	ParentID           string     `json:"parent_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	FirstCompletionAt  *time.Time `json:"first_completion_at,omitempty"`
}

// Node is a universal content unit — task, post, thread, or comment.
type Node struct {
	ID           string     `json:"id"`
	SpaceID      string     `json:"space_id"`
	ParentID     string     `json:"parent_id,omitempty"`
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"`
	Priority     string     `json:"priority"`
	Assignee     string     `json:"assignee"`
	AssigneeID   string     `json:"assignee_id"`              // user ID — source of truth for assignment
	AssigneeKind string     `json:"assignee_kind"`            // "human" or "agent", resolved from users table
	Author       string     `json:"author"`
	AuthorID     string     `json:"author_id"`               // user ID — source of truth for identity
	AuthorKind   string     `json:"author_kind"`              // "human" or "agent"
	Tags         []string   `json:"tags"`
	Pinned        bool       `json:"pinned"`
	ReplyToID     string     `json:"reply_to_id,omitempty"`     // message this is a reply to
	ReplyToAuthor string     `json:"reply_to_author,omitempty"` // resolved at query time
	ReplyToBody   string     `json:"reply_to_body,omitempty"`   // resolved at query time
	QuoteOfID     string     `json:"quote_of_id,omitempty"`     // post this quotes
	QuoteOfAuthor string     `json:"quote_of_author,omitempty"` // resolved at query time
	QuoteOfTitle  string     `json:"quote_of_title,omitempty"`  // resolved at query time
	QuoteOfBody   string     `json:"quote_of_body,omitempty"`   // resolved at query time
	DueDate       *time.Time `json:"due_date,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Verdict      string     `json:"verdict"`      // "approve", "revise", "reject" — set by review op
	Rating       int        `json:"rating"`        // 1-5 quality score — set by review op
	ChildCount   int        `json:"child_count"`
	ChildDone    int        `json:"child_done"`
	BlockerCount int        `json:"blocker_count"`
	Causes       []string   `json:"causes"`
}

// Op is a recorded grammar operation.
type Op struct {
	ID        string          `json:"id"`
	SpaceID   string          `json:"space_id"`
	NodeID    string          `json:"node_id,omitempty"`
	NodeTitle string          `json:"node_title,omitempty"` // resolved from nodes table when available
	Actor     string          `json:"actor"`
	ActorID   string          `json:"actor_id"`   // user ID — source of truth for identity
	ActorKind string          `json:"actor_kind"`  // "human" or "agent", resolved from users table
	Op        string          `json:"op"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// Reaction is a single emoji reaction on a node.
type Reaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Users []string `json:"users"` // user IDs who reacted with this emoji
}

// ────────────────────────────────────────────────────────────────────
// Params
// ────────────────────────────────────────────────────────────────────

// CreateNodeParams holds fields for creating a node.
type CreateNodeParams struct {
	SpaceID    string
	ParentID   string
	Kind       string
	Title      string
	Body       string
	State      string
	Priority   string
	Assignee   string
	AssigneeID string // user ID — source of truth for assignment
	Author     string
	AuthorID   string // user ID — source of truth for identity
	AuthorKind string // "human" or "agent"
	Tags       []string
	DueDate    *time.Time
	ReplyToID  string
	QuoteOfID  string
	Causes     []string
}

// ListNodesParams controls filtering for node listing.
type ListNodesParams struct {
	SpaceID  string
	Kind     string     // filter by kind, empty = all
	State    string     // filter by state, empty = all
	ParentID string     // "root" = top-level only, ID = children, empty = all
	Query    string     // ILIKE search on title/body, empty = all
	After    *time.Time // only nodes created after this time, nil = all
	Pinned   bool       // if true, only return pinned nodes
	CausedBy string     // if set, return nodes where this ID is in their causes array
	Limit    int        // max results, 0 = default (500)
}

// ────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────

var (
	ErrNotFound            = errors.New("not found")
	ErrChildrenIncomplete  = errors.New("cannot complete task: incomplete children")
)

// ────────────────────────────────────────────────────────────────────
// Store
// ────────────────────────────────────────────────────────────────────

// Store is a Postgres-backed store for the unified product.
// OpSubscriber is called after every op is recorded. Fire-and-forget —
// subscribers should not block. This is the pub/sub channel.
type OpSubscriber func(op *Op)

type Store struct {
	db          *sql.DB
	subscribers []OpSubscriber
}

// OnOp registers a subscriber that fires after every recorded op.
// Used by the hive to react to graph events in real time.
func (s *Store) OnOp(fn OpSubscriber) {
	s.subscribers = append(s.subscribers, fn)
}

// WebhookSubscriber returns an OpSubscriber that POSTs each op as JSON
// to the given URL. Fire-and-forget with a 5-second timeout.
func WebhookSubscriber(url string) OpSubscriber {
	client := &http.Client{Timeout: 5 * time.Second}
	return func(op *Op) {
		data, err := json.Marshal(op)
		if err != nil {
			return
		}
		resp, err := client.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			return
		}
		resp.Body.Close()
	}
}

// NewStore wraps an existing database connection and runs migrations.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("graph migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS spaces (
    id          TEXT PRIMARY KEY,
    slug        TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    owner_id    TEXT NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'project',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nodes (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_id   TEXT REFERENCES nodes(id),
    kind        TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    body        TEXT NOT NULL DEFAULT '',
    state       TEXT NOT NULL DEFAULT 'open',
    priority    TEXT NOT NULL DEFAULT 'medium',
    assignee    TEXT NOT NULL DEFAULT '',
    author      TEXT NOT NULL DEFAULT '',
    author_kind TEXT NOT NULL DEFAULT 'human',
    tags        TEXT[] NOT NULL DEFAULT '{}',
    due_date    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ops (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    node_id     TEXT REFERENCES nodes(id) ON DELETE CASCADE,
    actor       TEXT NOT NULL,
    op          TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nodes_space ON nodes(space_id);
CREATE INDEX IF NOT EXISTS idx_nodes_parent ON nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_nodes_kind ON nodes(space_id, kind);
CREATE INDEX IF NOT EXISTS idx_nodes_state ON nodes(space_id, state);
CREATE INDEX IF NOT EXISTS idx_ops_space ON ops(space_id);
CREATE INDEX IF NOT EXISTS idx_ops_node ON ops(node_id);
CREATE INDEX IF NOT EXISTS idx_ops_op ON ops(space_id, op);

ALTER TABLE spaces ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'private';
ALTER TABLE spaces ADD COLUMN IF NOT EXISTS parent_id TEXT;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS author_kind TEXT NOT NULL DEFAULT 'human';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS author_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS node_deps (
    node_id    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    depends_on TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (node_id, depends_on)
);
CREATE INDEX IF NOT EXISTS idx_node_deps_node ON node_deps(node_id);
CREATE INDEX IF NOT EXISTS idx_node_deps_dep ON node_deps(depends_on);
ALTER TABLE ops ADD COLUMN IF NOT EXISTS actor_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS mind_state (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS space_members (
    space_id   TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    user_name  TEXT NOT NULL DEFAULT '',
    joined_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (space_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_space_members_user ON space_members(user_id);

ALTER TABLE nodes ADD COLUMN IF NOT EXISTS assignee_id TEXT NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS pinned BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS reply_to_id TEXT NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS quote_of_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS read_state (
    user_id         TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    last_read_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, conversation_id)
);

CREATE TABLE IF NOT EXISTS endorsements (
    from_id    TEXT NOT NULL,
    to_id      TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (from_id, to_id)
);
CREATE INDEX IF NOT EXISTS idx_endorsements_to ON endorsements(to_id);

CREATE TABLE IF NOT EXISTS follows (
    follower_id TEXT NOT NULL,
    followed_id TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (follower_id, followed_id)
);
CREATE INDEX IF NOT EXISTS idx_follows_followed ON follows(followed_id);

CREATE TABLE IF NOT EXISTS reposts (
    user_id  TEXT NOT NULL,
    node_id  TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, node_id)
);
CREATE INDEX IF NOT EXISTS idx_reposts_node ON reposts(node_id);

ALTER TABLE nodes ADD COLUMN IF NOT EXISTS verdict TEXT NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS rating INT NOT NULL DEFAULT 0;
ALTER TABLE spaces ADD COLUMN IF NOT EXISTS first_completion_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS notifications (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    op_id      TEXT NOT NULL,
    space_id   TEXT NOT NULL,
    message    TEXT NOT NULL DEFAULT '',
    read       BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, read, created_at DESC);

CREATE TABLE IF NOT EXISTS invites (
    token    TEXT PRIMARY KEY,
    space_id TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reactions (
    node_id    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    emoji      TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (node_id, user_id, emoji)
);
CREATE INDEX IF NOT EXISTS idx_reactions_node ON reactions(node_id);

-- users table is created by auth.Auth.migrate(). Graph queries JOIN on it.
-- Creating here too (IF NOT EXISTS) ensures tests work without auth setup.
CREATE TABLE IF NOT EXISTS users (
    id         TEXT PRIMARY KEY,
    google_id  TEXT UNIQUE NOT NULL,
    email      TEXT UNIQUE NOT NULL,
    name       TEXT NOT NULL,
    picture    TEXT NOT NULL DEFAULT '',
    kind       TEXT NOT NULL DEFAULT 'human',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS reputation_score INT NOT NULL DEFAULT 0;
ALTER TABLE space_members ADD COLUMN IF NOT EXISTS welcomed_at TIMESTAMPTZ;

-- Backfill assignee_id from users table where assignee name matches.
-- (Placed after CREATE TABLE users to avoid reference errors on fresh databases.)
UPDATE nodes SET assignee_id = u.id
FROM users u WHERE nodes.assignee = u.name AND nodes.assignee_id = '' AND nodes.assignee != '';

-- api_keys is created by auth.Auth.migrate(). Defined here too (IF NOT EXISTS)
-- so GetHiveAgentID can query it in tests without requiring the full auth setup.
CREATE TABLE IF NOT EXISTS api_keys (
    id         TEXT PRIMARY KEY,
    key_hash   TEXT UNIQUE NOT NULL DEFAULT '',
    user_id    TEXT NOT NULL DEFAULT '',
    agent_id   TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agent_personas (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    display     TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT 'general',
    prompt      TEXT NOT NULL DEFAULT '',
    model       TEXT NOT NULL DEFAULT 'sonnet',
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agent_memories (
    id          TEXT PRIMARY KEY,
    persona     TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    memory      TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_agent_memories_lookup ON agent_memories(persona, user_id);

ALTER TABLE nodes ADD COLUMN IF NOT EXISTS last_message_preview TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS persona_name TEXT;
ALTER TABLE agent_memories ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'context';
ALTER TABLE agent_memories ADD COLUMN IF NOT EXISTS source_id TEXT NOT NULL DEFAULT '';
ALTER TABLE agent_memories ADD COLUMN IF NOT EXISTS importance INT NOT NULL DEFAULT 5;
ALTER TABLE agent_memories ADD COLUMN IF NOT EXISTS space_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_agent_memories_space ON agent_memories(space_id, user_id, persona);
ALTER TABLE agent_personas ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ;
ALTER TABLE agent_personas ADD COLUMN IF NOT EXISTS session_id UUID DEFAULT gen_random_uuid();
ALTER TABLE agent_personas ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ready';

CREATE TABLE IF NOT EXISTS invite_uses (
    token   TEXT NOT NULL REFERENCES invites(token) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    used_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (token, user_id)
);
ALTER TABLE invites ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE invites ADD COLUMN IF NOT EXISTS max_uses INT NOT NULL DEFAULT 0;
ALTER TABLE invites ADD COLUMN IF NOT EXISTS use_count INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS node_members (
    node_id    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    joined_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (node_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_node_members_user ON node_members(user_id);
ALTER TABLE node_members DROP COLUMN IF EXISTS user_name;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS causes TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS quorum_pct INT NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS voting_body TEXT NOT NULL DEFAULT 'all';

CREATE TABLE IF NOT EXISTS delegations (
    space_id     TEXT NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    delegator_id TEXT NOT NULL,
    delegate_id  TEXT NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (space_id, delegator_id)
);
CREATE INDEX IF NOT EXISTS idx_delegations_delegate ON delegations(space_id, delegate_id);

-- hive_diagnostics stores phase events from the hive runner so the /hive/feed
-- endpoint works in production (Fly) where loop/diagnostics.jsonl is not present.
CREATE TABLE IF NOT EXISTS hive_diagnostics (
    id         TEXT PRIMARY KEY,
    phase      TEXT NOT NULL DEFAULT '',
    outcome    TEXT NOT NULL DEFAULT '',
    cost_usd   FLOAT8 NOT NULL DEFAULT 0,
    payload    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_hive_diagnostics_created ON hive_diagnostics(created_at DESC);
`)
	return err
}

// ────────────────────────────────────────────────────────────────────
// Spaces
// ────────────────────────────────────────────────────────────────────

// CreateSpace creates a new space.
func (s *Store) CreateSpace(ctx context.Context, slug, name, description, ownerID, kind, visibility string) (*Space, error) {
	if kind == "" {
		kind = SpaceProject
	}
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	sp := &Space{
		ID:          newID(),
		Slug:        slug,
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		Kind:        kind,
		Visibility:  visibility,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO spaces (id, slug, name, description, owner_id, kind, visibility) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`,
		sp.ID, sp.Slug, sp.Name, sp.Description, sp.OwnerID, sp.Kind, sp.Visibility,
	).Scan(&sp.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create space: %w", err)
	}
	return sp, nil
}

// ListSpaces returns spaces owned by the given user.
func (s *Store) ListSpaces(ctx context.Context, ownerID string) ([]Space, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, slug, name, description, owner_id, kind, visibility, COALESCE(parent_id, ''), created_at
		 FROM spaces WHERE owner_id = $1 ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}
	defer rows.Close()

	var spaces []Space
	for rows.Next() {
		var sp Space
		if err := rows.Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.ParentID, &sp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan space: %w", err)
		}
		spaces = append(spaces, sp)
	}
	return spaces, rows.Err()
}

// ListChildSpaces returns spaces whose parent_id matches the given space ID.
func (s *Store) ListChildSpaces(ctx context.Context, parentID string) ([]Space, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, slug, name, description, owner_id, kind, visibility, COALESCE(parent_id, ''), created_at
		 FROM spaces WHERE parent_id = $1 ORDER BY name`, parentID)
	if err != nil {
		return nil, fmt.Errorf("list child spaces: %w", err)
	}
	defer rows.Close()
	var spaces []Space
	for rows.Next() {
		var sp Space
		if err := rows.Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.ParentID, &sp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan space: %w", err)
		}
		spaces = append(spaces, sp)
	}
	return spaces, rows.Err()
}

// SpaceWithStats extends Space with aggregate counts for the discover page.
type SpaceWithStats struct {
	Space
	NodeCount    int        `json:"node_count"`
	LastActivity *time.Time `json:"last_activity,omitempty"`
	MemberCount  int        `json:"member_count"`
	HasAgent     bool       `json:"has_agent"`
}

// ListPublicSpaces returns all public spaces with node counts, last activity,
// member count, and agent presence.
func (s *Store) ListPublicSpaces(ctx context.Context, query ...string) ([]SpaceWithStats, error) {
	q := `SELECT s.id, s.slug, s.name, s.description, s.owner_id, s.kind, s.visibility, COALESCE(s.parent_id, ''), s.created_at,
		        COALESCE(n.cnt, 0), n.last_at,
		        COALESCE(m.member_count, 0), COALESCE(m.has_agent, false)
		 FROM spaces s
		 LEFT JOIN LATERAL (
		     SELECT COUNT(*) AS cnt, MAX(created_at) AS last_at
		     FROM nodes WHERE space_id = s.id
		 ) n ON true
		 LEFT JOIN LATERAL (
		     SELECT COUNT(DISTINCT o.actor_id) AS member_count,
		            BOOL_OR(u.kind = 'agent') AS has_agent
		     FROM ops o
		     LEFT JOIN users u ON u.id = o.actor_id
		     WHERE o.space_id = s.id
		 ) m ON true
		 WHERE s.visibility = 'public' AND s.parent_id IS NULL`
	var args []any
	if len(query) > 0 && query[0] != "" {
		q += ` AND (s.name ILIKE $1 OR s.description ILIKE $1)`
		args = append(args, "%"+query[0]+"%")
	}
	q += ` ORDER BY COALESCE(n.last_at, s.created_at) DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list public spaces: %w", err)
	}
	defer rows.Close()

	var spaces []SpaceWithStats
	for rows.Next() {
		var sp SpaceWithStats
		var lastAt sql.NullTime
		if err := rows.Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.ParentID, &sp.CreatedAt,
			&sp.NodeCount, &lastAt, &sp.MemberCount, &sp.HasAgent); err != nil {
			return nil, fmt.Errorf("scan space: %w", err)
		}
		if lastAt.Valid {
			sp.LastActivity = &lastAt.Time
		}
		spaces = append(spaces, sp)
	}
	return spaces, rows.Err()
}

// UpdateSpace updates a space's mutable fields.
func (s *Store) UpdateSpace(ctx context.Context, id, name, description, visibility string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE spaces SET name = $1, description = $2, visibility = $3 WHERE id = $4`,
		name, description, visibility, id,
	)
	if err != nil {
		return fmt.Errorf("update space: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSpace removes a space and all its nodes and ops (via CASCADE).
func (s *Store) DeleteSpace(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM spaces WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("delete space: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSpaceByID returns a space by its ID.
func (s *Store) GetSpaceByID(ctx context.Context, id string) (*Space, error) {
	var sp Space
	err := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, description, owner_id, kind, visibility, COALESCE(parent_id, ''), created_at, first_completion_at FROM spaces WHERE id = $1`, id,
	).Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.ParentID, &sp.CreatedAt, &sp.FirstCompletionAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get space: %w", err)
	}
	return &sp, nil
}

// GetSpaceBySlug returns a space by its slug.
func (s *Store) GetSpaceBySlug(ctx context.Context, slug string) (*Space, error) {
	var sp Space
	err := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, description, owner_id, kind, visibility, COALESCE(parent_id, ''), created_at, first_completion_at FROM spaces WHERE slug = $1`, slug,
	).Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.ParentID, &sp.CreatedAt, &sp.FirstCompletionAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get space: %w", err)
	}
	return &sp, nil
}

// MarkFirstCompletion sets first_completion_at on a space if it hasn't been set yet.
// Returns true if this was the first completion (i.e., the column was NULL before).
func (s *Store) MarkFirstCompletion(ctx context.Context, spaceID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE spaces SET first_completion_at = NOW() WHERE id = $1 AND first_completion_at IS NULL`,
		spaceID,
	)
	if err != nil {
		return false, fmt.Errorf("mark first completion: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ────────────────────────────────────────────────────────────────────
// Nodes
// ────────────────────────────────────────────────────────────────────

// CreateNode creates a new node in a space.
func (s *Store) CreateNode(ctx context.Context, p CreateNodeParams) (*Node, error) {
	if p.State == "" {
		p.State = StateOpen
	}
	if p.Priority == "" {
		p.Priority = PriorityMedium
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}

	authorKind := p.AuthorKind
	if authorKind == "" {
		authorKind = "human"
	}

	causes := p.Causes
	if causes == nil {
		causes = []string{}
	}
	n := &Node{
		ID:         newID(),
		SpaceID:    p.SpaceID,
		ParentID:   p.ParentID,
		Kind:       p.Kind,
		Title:      p.Title,
		Body:       p.Body,
		State:      p.State,
		Priority:   p.Priority,
		Assignee:   p.Assignee,
		AssigneeID: p.AssigneeID,
		Author:     p.Author,
		AuthorID:   p.AuthorID,
		AuthorKind: authorKind,
		Tags:       p.Tags,
		DueDate:    p.DueDate,
		ReplyToID:  p.ReplyToID,
		QuoteOfID:  p.QuoteOfID,
		Causes:     causes,
	}

	var parentID *string
	if p.ParentID != "" {
		parentID = &p.ParentID
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO nodes (id, space_id, parent_id, kind, title, body, state, priority, assignee, assignee_id, author, author_id, author_kind, tags, due_date, reply_to_id, quote_of_id, causes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		 RETURNING created_at, updated_at`,
		n.ID, n.SpaceID, parentID, n.Kind, n.Title, n.Body, n.State, n.Priority,
		n.Assignee, n.AssigneeID, n.Author, n.AuthorID, n.AuthorKind, pq.Array(n.Tags), n.DueDate, n.ReplyToID, n.QuoteOfID, pq.Array(n.Causes),
	).Scan(&n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create node: %w", err)
	}
	return n, nil
}

// GetNode returns a node by ID with child counts.
func (s *Store) GetNode(ctx context.Context, id string) (*Node, error) {
	var n Node
	var parentID sql.NullString
	var dueDate sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT n.id, n.space_id, n.parent_id, n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind, n.tags, n.pinned, n.due_date,
		       n.created_at, n.updated_at, n.verdict, n.rating,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM node_deps d JOIN nodes b ON b.id = d.depends_on WHERE d.node_id = n.id AND b.state != 'done'), 0),
		       n.reply_to_id,
		       COALESCE((SELECT r.author FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       COALESCE((SELECT LEFT(r.body, 80) FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       n.quote_of_id,
		       COALESCE((SELECT q.author FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT q.title FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT LEFT(q.body, 120) FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE(au.kind, ''), n.causes
		FROM nodes n
		LEFT JOIN users au ON au.id = n.assignee_id
		WHERE n.id = $1`, id,
	).Scan(
		&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
		&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind, pq.Array(&n.Tags), &n.Pinned, &dueDate,
		&n.CreatedAt, &n.UpdatedAt, &n.Verdict, &n.Rating,
		&n.ChildCount, &n.ChildDone, &n.BlockerCount,
		&n.ReplyToID, &n.ReplyToAuthor, &n.ReplyToBody,
		&n.QuoteOfID, &n.QuoteOfAuthor, &n.QuoteOfTitle, &n.QuoteOfBody,
		&n.AssigneeKind, pq.Array(&n.Causes),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	if parentID.Valid {
		n.ParentID = parentID.String
	}
	if dueDate.Valid {
		d := dueDate.Time
		n.DueDate = &d
	}
	return &n, nil
}

// ListNodes returns nodes matching the filter criteria.
func (s *Store) ListNodes(ctx context.Context, p ListNodesParams) ([]Node, error) {
	query := `
		SELECT n.id, n.space_id, n.parent_id, n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind, n.tags, n.pinned, n.due_date,
		       n.created_at, n.updated_at, n.verdict, n.rating,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM node_deps d JOIN nodes b ON b.id = d.depends_on WHERE d.node_id = n.id AND b.state != 'done'), 0),
		       n.reply_to_id,
		       COALESCE((SELECT r.author FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       COALESCE((SELECT LEFT(r.body, 80) FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       n.quote_of_id,
		       COALESCE((SELECT q.author FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT q.title FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT LEFT(q.body, 120) FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE(au.kind, ''), n.causes
		FROM nodes n
		LEFT JOIN users au ON au.id = n.assignee_id
		WHERE n.space_id = $1`

	args := []any{p.SpaceID}
	argN := 2

	if p.Kind != "" {
		query += fmt.Sprintf(" AND n.kind = $%d", argN)
		args = append(args, p.Kind)
		argN++
	}
	if p.State != "" {
		query += fmt.Sprintf(" AND n.state = $%d", argN)
		args = append(args, p.State)
		argN++
	}
	if p.ParentID == "root" {
		query += " AND n.parent_id IS NULL"
	} else if p.ParentID != "" {
		query += fmt.Sprintf(" AND n.parent_id = $%d", argN)
		args = append(args, p.ParentID)
		argN++
	}
	if p.After != nil {
		query += fmt.Sprintf(" AND n.created_at > $%d", argN)
		args = append(args, *p.After)
		argN++
	}
	if p.Pinned {
		query += " AND n.pinned = true"
	}
	if p.CausedBy != "" {
		query += fmt.Sprintf(" AND $%d = ANY(n.causes)", argN)
		args = append(args, p.CausedBy)
		argN++
	}
	if p.Query != "" {
		query += fmt.Sprintf(" AND (n.title ILIKE '%%' || $%d || '%%' OR n.body ILIKE '%%' || $%d || '%%')", argN, argN)
		args = append(args, p.Query)
		argN++
	}

	query += " ORDER BY n.pinned DESC, n.created_at DESC"

	limit := p.Limit
	if limit <= 0 {
		limit = 500
	}
	query += fmt.Sprintf(" LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime

		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind, pq.Array(&n.Tags), &n.Pinned, &dueDate,
			&n.CreatedAt, &n.UpdatedAt, &n.Verdict, &n.Rating,
			&n.ChildCount, &n.ChildDone, &n.BlockerCount,
			&n.ReplyToID, &n.ReplyToAuthor, &n.ReplyToBody,
			&n.QuoteOfID, &n.QuoteOfAuthor, &n.QuoteOfTitle, &n.QuoteOfBody,
			&n.AssigneeKind, pq.Array(&n.Causes),
		); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}

		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ListPostsByEngagement returns posts sorted by an engagement score:
// endorsements * 3 + reposts * 2 + replies + recency bonus (7 - days_old, min 0).
func (s *Store) ListPostsByEngagement(ctx context.Context, spaceID string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT n.id, n.space_id, n.parent_id, n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind, n.tags, n.pinned, n.due_date,
		       n.created_at, n.updated_at, n.verdict, n.rating,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM node_deps d JOIN nodes b ON b.id = d.depends_on WHERE d.node_id = n.id AND b.state != 'done'), 0),
		       n.reply_to_id,
		       COALESCE((SELECT r.author FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       COALESCE((SELECT LEFT(r.body, 80) FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       n.quote_of_id,
		       COALESCE((SELECT q.author FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT q.title FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT LEFT(q.body, 120) FROM nodes q WHERE q.id = n.quote_of_id), '')
		FROM nodes n
		WHERE n.space_id = $1 AND n.kind = 'post' AND n.parent_id IS NULL
		ORDER BY
		  COALESCE((SELECT COUNT(*) FROM endorsements e WHERE e.to_id = n.id), 0) * 3
		  + COALESCE((SELECT COUNT(*) FROM reposts rp WHERE rp.node_id = n.id), 0) * 2
		  + COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0)
		  + GREATEST(0, 7 - EXTRACT(DAY FROM NOW() - n.created_at))
		  DESC,
		  n.created_at DESC
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, spaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list posts by engagement: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime

		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind, pq.Array(&n.Tags), &n.Pinned, &dueDate,
			&n.CreatedAt, &n.UpdatedAt, &n.Verdict, &n.Rating,
			&n.ChildCount, &n.ChildDone, &n.BlockerCount,
			&n.ReplyToID, &n.ReplyToAuthor, &n.ReplyToBody,
			&n.QuoteOfID, &n.QuoteOfAuthor, &n.QuoteOfTitle, &n.QuoteOfBody,
		); err != nil {
			return nil, fmt.Errorf("scan engagement post: %w", err)
		}

		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ListPostsByTrending returns posts sorted by engagement velocity:
// recent engagement (last 48h) divided by post age in hours.
func (s *Store) ListPostsByTrending(ctx context.Context, spaceID string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT n.id, n.space_id, n.parent_id, n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind, n.tags, n.pinned, n.due_date,
		       n.created_at, n.updated_at, n.verdict, n.rating,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM node_deps d JOIN nodes b ON b.id = d.depends_on WHERE d.node_id = n.id AND b.state != 'done'), 0),
		       n.reply_to_id,
		       COALESCE((SELECT r.author FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       COALESCE((SELECT LEFT(r.body, 80) FROM nodes r WHERE r.id = n.reply_to_id), ''),
		       n.quote_of_id,
		       COALESCE((SELECT q.author FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT q.title FROM nodes q WHERE q.id = n.quote_of_id), ''),
		       COALESCE((SELECT LEFT(q.body, 120) FROM nodes q WHERE q.id = n.quote_of_id), '')
		FROM nodes n
		WHERE n.space_id = $1 AND n.kind = 'post' AND n.parent_id IS NULL
		ORDER BY
		  (
		    COALESCE((SELECT COUNT(*) FROM endorsements e WHERE e.to_id = n.id AND e.created_at > NOW() - INTERVAL '48 hours'), 0) * 3
		    + COALESCE((SELECT COUNT(*) FROM reposts rp WHERE rp.node_id = n.id AND rp.created_at > NOW() - INTERVAL '48 hours'), 0) * 2
		    + COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.created_at > NOW() - INTERVAL '48 hours'), 0)
		  )::float / GREATEST(1, EXTRACT(EPOCH FROM NOW() - n.created_at) / 3600)
		  DESC,
		  n.created_at DESC
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, spaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list posts by trending: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime

		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind, pq.Array(&n.Tags), &n.Pinned, &dueDate,
			&n.CreatedAt, &n.UpdatedAt, &n.Verdict, &n.Rating,
			&n.ChildCount, &n.ChildDone, &n.BlockerCount,
			&n.ReplyToID, &n.ReplyToAuthor, &n.ReplyToBody,
			&n.QuoteOfID, &n.QuoteOfAuthor, &n.QuoteOfTitle, &n.QuoteOfBody,
		); err != nil {
			return nil, fmt.Errorf("scan trending post: %w", err)
		}

		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ConversationSummary is a conversation node with last message preview.
type ConversationSummary struct {
	Node
	LastAuthor     string `json:"last_author,omitempty"`
	LastAuthorKind string `json:"last_author_kind,omitempty"`
	LastBody       string `json:"last_body,omitempty"`
	UnreadCount    int    `json:"unread_count"`
}

// ListConversations returns conversations in a space that involve the given user.
// Matches on userID in tags or as author_id.
func (s *Store) ListConversations(ctx context.Context, spaceID, userID string) ([]ConversationSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, n.parent_id, n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind, n.tags, n.pinned, n.due_date,
		       n.created_at, n.updated_at, n.verdict, n.rating,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       0, 0,
		       lm.author, lm.author_kind,
		       COALESCE(NULLIF(n.last_message_preview, ''), lm.body),
		       COALESCE((
		         SELECT COUNT(*) FROM nodes c
		         WHERE c.parent_id = n.id
		           AND c.created_at > COALESCE(
		             (SELECT rs.last_read_at FROM read_state rs WHERE rs.user_id = $2 AND rs.conversation_id = n.id),
		             '1970-01-01'::timestamptz)
		       ), 0)
		FROM nodes n
		LEFT JOIN LATERAL (
		    SELECT c.author, c.author_kind, c.body
		    FROM nodes c WHERE c.parent_id = n.id
		    ORDER BY c.created_at DESC LIMIT 1
		) lm ON true
		WHERE n.space_id = $1 AND n.kind = 'conversation'
		  AND ($2 = ANY(n.tags) OR n.author_id = $2)
		ORDER BY n.updated_at DESC
		LIMIT 100`, spaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var convos []ConversationSummary
	for rows.Next() {
		var cs ConversationSummary
		var parentID sql.NullString
		var dueDate sql.NullTime
		var lastAuthor, lastAuthorKind, lastBody sql.NullString
		if err := rows.Scan(
			&cs.ID, &cs.SpaceID, &parentID, &cs.Kind, &cs.Title, &cs.Body,
			&cs.State, &cs.Priority, &cs.Assignee, &cs.AssigneeID, &cs.Author, &cs.AuthorID, &cs.AuthorKind, pq.Array(&cs.Tags), &cs.Pinned, &dueDate,
			&cs.CreatedAt, &cs.UpdatedAt, &cs.Verdict, &cs.Rating,
			&cs.ChildCount, &cs.ChildDone, &cs.BlockerCount,
			&lastAuthor, &lastAuthorKind, &lastBody,
			&cs.UnreadCount,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		if parentID.Valid {
			cs.ParentID = parentID.String
		}
		if lastAuthor.Valid {
			cs.LastAuthor = lastAuthor.String
		}
		if lastAuthorKind.Valid {
			cs.LastAuthorKind = lastAuthorKind.String
		}
		if lastBody.Valid {
			cs.LastBody = lastBody.String
		}
		convos = append(convos, cs)
	}
	return convos, rows.Err()
}

// UpdateLastMessagePreview stores the first 100 chars of the latest message body
// on the conversation node to avoid a lateral join in list queries.
func (s *Store) UpdateLastMessagePreview(ctx context.Context, conversationID, body string) error {
	runes := []rune(body)
	if len(runes) > 100 {
		runes = runes[:100]
	}
	preview := string(runes)
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET last_message_preview = $1, updated_at = NOW() WHERE id = $2`,
		preview, conversationID)
	return err
}

// ResolveUserID returns the user ID for a given name, or empty string if not found.
func (s *Store) ResolveUserID(ctx context.Context, name string) string {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE name = $1`, name).Scan(&id)
	if err != nil {
		return ""
	}
	return id
}

// ResolveUserNames maps user IDs to display names. Unknown IDs are returned as-is.
func (s *Store) ResolveUserNames(ctx context.Context, ids []string) map[string]string {
	result := make(map[string]string, len(ids))
	for _, id := range ids {
		result[id] = id // default to showing the ID
	}
	if len(ids) == 0 {
		return result
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM users WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if rows.Scan(&id, &name) == nil {
			result[id] = name
		}
	}
	return result
}

// ListAgentNames returns names of all agent users.
func (s *Store) ListAgentNames(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM users WHERE kind = 'agent' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list agent names: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// SearchUsers returns users whose names match a query (ILIKE), limited to 8.
func (s *Store) SearchUsers(ctx context.Context, query string) ([]struct{ Name, Kind string }, error) {
	if query == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, kind FROM users WHERE name ILIKE '%' || $1 || '%' ORDER BY name LIMIT 8`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []struct{ Name, Kind string }
	for rows.Next() {
		var u struct{ Name, Kind string }
		if err := rows.Scan(&u.Name, &u.Kind); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetFirstAgent returns the id and name of the first agent user, or ("", "", nil) if none exists.
func (s *Store) GetFirstAgent(ctx context.Context) (string, string, error) {
	var id, name string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name FROM users WHERE kind = 'agent' ORDER BY name LIMIT 1`,
	).Scan(&id, &name)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return id, name, err
}

// HasAgentParticipant checks if any of the given IDs belong to an agent user.
// Tags now store user IDs, so we match on id.
func (s *Store) HasAgentParticipant(ctx context.Context, ids []string) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = ANY($1) AND kind = 'agent')`,
		pq.Array(ids)).Scan(&exists)
	return exists, err
}

// ListDocumentContext returns KindDocument nodes in a space for prompt context injection.
// Returns at most 10 documents with title and body (BOUNDED invariant).
func (s *Store) ListDocumentContext(ctx context.Context, spaceID string) ([]Node, error) {
	return s.ListNodes(ctx, ListNodesParams{
		SpaceID: spaceID,
		Kind:    KindDocument,
		Limit:   10,
	})
}

// ListDocuments returns KindDocument nodes in a space (BOUNDED: hard limit).
func (s *Store) ListDocuments(ctx context.Context, spaceID string, limit int) ([]Node, error) {
	return s.ListNodes(ctx, ListNodesParams{
		SpaceID: spaceID,
		Kind:    KindDocument,
		Limit:   limit,
	})
}

// ListQuestions returns KindQuestion nodes in a space (BOUNDED: hard limit).
func (s *Store) ListQuestions(ctx context.Context, spaceID string, limit int) ([]Node, error) {
	return s.ListNodes(ctx, ListNodesParams{
		SpaceID: spaceID,
		Kind:    KindQuestion,
		Limit:   limit,
	})
}

// ListCouncilSessions returns KindCouncil nodes in a space (BOUNDED: hard limit).
func (s *Store) ListCouncilSessions(ctx context.Context, spaceID string, limit int) ([]Node, error) {
	return s.ListNodes(ctx, ListNodesParams{
		SpaceID: spaceID,
		Kind:    KindCouncil,
		Limit:   limit,
	})
}

// maxCascadeDepth bounds the recursive child-close to satisfy invariant 13 (BOUNDED).
const maxCascadeDepth = 50

// cascadeCloseChildren sets all non-done descendants of parentID to done.
// Depth-first: grandchildren are closed before their parents.
func (s *Store) cascadeCloseChildren(ctx context.Context, parentID string, depth int) error {
	if depth > maxCascadeDepth {
		return fmt.Errorf("cascade depth exceeded for parent %s", parentID)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM nodes WHERE parent_id = $1 AND state != 'done'`, parentID,
	)
	if err != nil {
		return fmt.Errorf("cascade children of %s: %w", parentID, err)
	}
	var childIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		childIDs = append(childIDs, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, childID := range childIDs {
		if err := s.cascadeCloseChildren(ctx, childID, depth+1); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE nodes SET state = 'done', updated_at = NOW() WHERE id = $1`,
			childID,
		); err != nil {
			return fmt.Errorf("cascade close child %s: %w", childID, err)
		}
	}
	return nil
}

// UpdateNodeState sets a node's state.
// When transitioning to done, all non-done descendants are auto-closed first.
func (s *Store) UpdateNodeState(ctx context.Context, id, state string) error {
	if state == StateDone {
		if err := s.cascadeCloseChildren(ctx, id, 0); err != nil {
			return err
		}
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET state = $1, updated_at = NOW() WHERE id = $2`,
		state, id,
	)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ClaimNode sets the assignee and transitions state to active in one atomic operation.
func (s *Store) ClaimNode(ctx context.Context, nodeID, userName, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET assignee = $1, assignee_id = $2, state = $3, updated_at = NOW()
		 WHERE id = $4 AND (assignee_id = '' OR assignee_id = $2)`,
		userName, userID, StateActive, nodeID,
	)
	if err != nil {
		return fmt.Errorf("claim node: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("task already claimed by someone else")
	}
	return nil
}

// UpdateNode updates mutable fields on a node.
func (s *Store) UpdateNode(ctx context.Context, id string, title, body, priority, assignee, assigneeID *string) error {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	argN := 1

	if title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argN))
		args = append(args, *title)
		argN++
	}
	if body != nil {
		sets = append(sets, fmt.Sprintf("body = $%d", argN))
		args = append(args, *body)
		argN++
	}
	if priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argN))
		args = append(args, *priority)
		argN++
	}
	if assignee != nil {
		sets = append(sets, fmt.Sprintf("assignee = $%d", argN))
		args = append(args, *assignee)
		argN++
	}
	if assigneeID != nil {
		sets = append(sets, fmt.Sprintf("assignee_id = $%d", argN))
		args = append(args, *assigneeID)
		argN++
	}

	query := fmt.Sprintf("UPDATE nodes SET %s WHERE id = $%d",
		joinStrings(sets, ", "), argN)
	args = append(args, id)

	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update node: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteNode removes a node. Children cascade.
func (s *Store) DeleteNode(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM nodes WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────
// Ops
// ────────────────────────────────────────────────────────────────────

// RecordOp records a grammar operation.
func (s *Store) RecordOp(ctx context.Context, spaceID, nodeID, actor, actorID, op string, payload json.RawMessage) (*Op, error) {
	if payload == nil {
		payload = json.RawMessage(`{}`)
	}
	o := &Op{
		ID:      newID(),
		SpaceID: spaceID,
		NodeID:  nodeID,
		Actor:   actor,
		ActorID: actorID,
		Op:      op,
		Payload: payload,
	}

	var nodeRef *string
	if nodeID != "" {
		nodeRef = &nodeID
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO ops (id, space_id, node_id, actor, actor_id, op, payload) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`,
		o.ID, o.SpaceID, nodeRef, o.Actor, o.ActorID, o.Op, o.Payload,
	).Scan(&o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("record op: %w", err)
	}

	// Notify subscribers (pub/sub). Fire-and-forget — don't block the op.
	for _, fn := range s.subscribers {
		go fn(o)
	}

	return o, nil
}

// ListOps returns recent operations for a space.
func (s *Store) ListOps(ctx context.Context, spaceID string, limit int) ([]Op, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.id, o.space_id, COALESCE(o.node_id, ''), COALESCE(n.title, ''),
		        o.actor, o.actor_id, COALESCE(u.kind, 'human'), o.op, o.payload, o.created_at
		 FROM ops o
		 LEFT JOIN users u ON u.id = o.actor_id
		 LEFT JOIN nodes n ON n.id = o.node_id
		 WHERE o.space_id = $1 ORDER BY o.created_at DESC LIMIT $2`,
		spaceID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list ops: %w", err)
	}
	defer rows.Close()

	var ops []Op
	for rows.Next() {
		var o Op
		if err := rows.Scan(&o.ID, &o.SpaceID, &o.NodeID, &o.NodeTitle, &o.Actor, &o.ActorID, &o.ActorKind, &o.Op, &o.Payload, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan op: %w", err)
		}
		ops = append(ops, o)
	}
	return ops, rows.Err()
}

// ListNodeOps returns operations for a specific node.
func (s *Store) ListNodeOps(ctx context.Context, nodeID string) ([]Op, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.id, o.space_id, COALESCE(o.node_id, ''), o.actor, o.actor_id,
		        COALESCE(u.kind, 'human'), o.op, o.payload, o.created_at
		 FROM ops o
		 LEFT JOIN users u ON u.id = o.actor_id
		 WHERE o.node_id = $1 ORDER BY o.created_at`,
		nodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("list node ops: %w", err)
	}
	defer rows.Close()

	var ops []Op
	for rows.Next() {
		var o Op
		if err := rows.Scan(&o.ID, &o.SpaceID, &o.NodeID, &o.Actor, &o.ActorID, &o.ActorKind, &o.Op, &o.Payload, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan op: %w", err)
		}
		ops = append(ops, o)
	}
	return ops, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

// PlatformStats returns aggregate counts for the platform.
type PlatformStats struct {
	Spaces     int
	Tasks      int
	Users      int
	AgentOps   int
}

func (s *Store) GetPlatformStats(ctx context.Context) PlatformStats {
	var stats PlatformStats
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM spaces WHERE visibility = 'public'`).Scan(&stats.Spaces)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE kind = 'task'`).Scan(&stats.Tasks)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.Users)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ops o JOIN users u ON u.id = o.actor_id WHERE u.kind = 'agent'`).Scan(&stats.AgentOps)
	return stats
}

// ────────────────────────────────────────────────────────────────────
// Space Membership
// ────────────────────────────────────────────────────────────────────

// JoinSpace adds a user as a member of a space.
func (s *Store) JoinSpace(ctx context.Context, spaceID, userID, userName string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO space_members (space_id, user_id, user_name) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		spaceID, userID, userName)
	return err
}

// LeaveSpace removes a user from a space.
func (s *Store) LeaveSpace(ctx context.Context, spaceID, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM space_members WHERE space_id = $1 AND user_id = $2`,
		spaceID, userID)
	return err
}

// IsMember checks if a user is a member of a space.
func (s *Store) IsMember(ctx context.Context, spaceID, userID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM space_members WHERE space_id = $1 AND user_id = $2)`,
		spaceID, userID).Scan(&exists)
	return exists
}

// MarkWelcomed sets welcomed_at for a member if not already set.
// Returns true only on the first call (when welcomed_at was NULL).
func (s *Store) MarkWelcomed(ctx context.Context, spaceID, userID string) bool {
	res, err := s.db.ExecContext(ctx,
		`UPDATE space_members SET welcomed_at = NOW()
		 WHERE space_id = $1 AND user_id = $2 AND welcomed_at IS NULL`,
		spaceID, userID)
	if err != nil {
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// SpaceMember holds a member's info for display.
type SpaceMember struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Kind     string `json:"kind"`
}

// ListMembers returns members of a space with their kind (human/agent).
func (s *Store) ListMembers(ctx context.Context, spaceID string, limit int) ([]SpaceMember, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT sm.user_id, sm.user_name, COALESCE(u.kind, 'human')
		 FROM space_members sm
		 LEFT JOIN users u ON u.id = sm.user_id
		 WHERE sm.space_id = $1
		 ORDER BY sm.joined_at
		 LIMIT $2`, spaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []SpaceMember
	for rows.Next() {
		var m SpaceMember
		if rows.Scan(&m.UserID, &m.UserName, &m.Kind) == nil {
			members = append(members, m)
		}
	}
	return members, rows.Err()
}

// NodeMember holds a node (team/role) member's info for display.
type NodeMember struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

// JoinNodeMember adds a user to a node (team/role) member list.
func (s *Store) JoinNodeMember(ctx context.Context, nodeID, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO node_members (node_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		nodeID, userID)
	return err
}

// LeaveNodeMember removes a user from a node (team/role) member list.
func (s *Store) LeaveNodeMember(ctx context.Context, nodeID, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM node_members WHERE node_id = $1 AND user_id = $2`,
		nodeID, userID)
	return err
}

// IsNodeMember checks if a user is a member of a node (team/role).
func (s *Store) IsNodeMember(ctx context.Context, nodeID, userID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM node_members WHERE node_id = $1 AND user_id = $2)`,
		nodeID, userID).Scan(&exists)
	return exists
}

// NodeMemberCount returns the number of members in a node (team/role).
func (s *Store) NodeMemberCount(ctx context.Context, nodeID string) int {
	var count int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM node_members WHERE node_id = $1`, nodeID).Scan(&count)
	return count
}

// ListTeamMembers returns members of a team node within a space.
// Display names are resolved from the users table at query time, not stored in node_members.
func (s *Store) ListTeamMembers(ctx context.Context, spaceID, teamID string) ([]NodeMember, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT nm.user_id, COALESCE(u.name, nm.user_id)
		 FROM node_members nm
		 JOIN nodes n ON n.id = nm.node_id
		 LEFT JOIN users u ON u.id = nm.user_id
		 WHERE nm.node_id = $1 AND n.space_id = $2
		 ORDER BY nm.joined_at
		 LIMIT 100`, teamID, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []NodeMember
	for rows.Next() {
		var m NodeMember
		if rows.Scan(&m.UserID, &m.UserName) == nil {
			members = append(members, m)
		}
	}
	return members, rows.Err()
}

// MemberCount returns the number of members in a space.
func (s *Store) MemberCount(ctx context.Context, spaceID string) int {
	var count int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM space_members WHERE space_id = $1`, spaceID).Scan(&count)
	return count
}

// GetUserProfile returns a user by name (for public profiles).
func (s *Store) GetUserProfile(ctx context.Context, name string) (*struct {
	ID              string
	Name            string
	Kind            string
	TasksDone       int
	OpCount         int
	ReputationScore int
}, error) {
	var u struct {
		ID              string
		Name            string
		Kind            string
		TasksDone       int
		OpCount         int
		ReputationScore int
	}
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.kind,
		       COALESCE((SELECT COUNT(*) FROM nodes n WHERE n.author_id = u.id AND n.kind = 'task' AND n.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM ops o WHERE o.actor_id = u.id), 0),
		       u.reputation_score
		FROM users u WHERE u.name = $1`, name,
	).Scan(&u.ID, &u.Name, &u.Kind, &u.TasksDone, &u.OpCount, &u.ReputationScore)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ComputeAndUpdateReputation recomputes a user's reputation score from their op
// history across all spaces and stores it in users.reputation_score.
// Formula: completed_tasks×1 + review_approvals×2 + review_revisions×0.5
//          + endorsements×1.5 - review_rejections×1
func (s *Store) ComputeAndUpdateReputation(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	var completedTasks int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM nodes WHERE assignee_id = $1 AND kind = 'task' AND state = 'done'`,
		userID).Scan(&completedTasks)

	// Count review verdicts on tasks this user was assignee of.
	var approvals, revisions, rejections int
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(o.payload->>'verdict', ''), COUNT(*)
		FROM ops o
		WHERE o.op = 'review'
		  AND o.node_id IN (SELECT id FROM nodes WHERE assignee_id = $1 AND kind = 'task')
		GROUP BY o.payload->>'verdict'`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var verdict string
			var count int
			if rows.Scan(&verdict, &count) == nil {
				switch verdict {
				case "approve":
					approvals = count
				case "revise":
					revisions = count
				case "reject":
					rejections = count
				}
			}
		}
	}

	var endorsements int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM endorsements WHERE to_id = $1`, userID).Scan(&endorsements)

	score := int(
		float64(completedTasks)*1.0 +
			float64(approvals)*2.0 +
			float64(revisions)*0.5 +
			float64(endorsements)*1.5 -
			float64(rejections)*1.0,
	)

	_, err = s.db.ExecContext(ctx,
		`UPDATE users SET reputation_score = $1 WHERE id = $2`, score, userID)
	return err
}

// GetReputationComponents returns the key stats shown on a profile: tasks completed
// (as assignee) and review approvals received. Used for "X tasks, Y approved" display.
func (s *Store) GetReputationComponents(ctx context.Context, userID string) (tasksCompleted, reviewApprovals int) {
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM nodes WHERE assignee_id = $1 AND kind = 'task' AND state = 'done'`,
		userID).Scan(&tasksCompleted)
	s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM ops o
		WHERE o.op = 'review' AND o.payload->>'verdict' = 'approve'
		  AND o.node_id IN (SELECT id FROM nodes WHERE assignee_id = $1 AND kind = 'task')`,
		userID).Scan(&reviewApprovals)
	return
}

// GetBulkReputationByIDs returns reputation_score for a set of user IDs.
func (s *Store) GetBulkReputationByIDs(ctx context.Context, userIDs []string) map[string]int {
	result := make(map[string]int)
	if len(userIDs) == 0 {
		return result
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, reputation_score FROM users WHERE id = ANY($1)`,
		pq.Array(userIDs))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var score int
		if rows.Scan(&id, &score) == nil {
			result[id] = score
		}
	}
	return result
}

// UserMembership is a space a user belongs to.
type UserMembership struct {
	SpaceSlug string
	SpaceName string
	SpaceKind string
}

// ListUserMemberships returns public spaces a user is a member of or owns.
func (s *Store) ListUserMemberships(ctx context.Context, userID string) ([]UserMembership, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.slug, s.name, s.kind FROM spaces s
		WHERE s.visibility = 'public'
		  AND (s.owner_id = $1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id = s.id AND sm.user_id = $1))
		ORDER BY s.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memberships []UserMembership
	for rows.Next() {
		var m UserMembership
		if err := rows.Scan(&m.SpaceSlug, &m.SpaceName, &m.SpaceKind); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}

// ListPublicActivity returns recent ops from public spaces.
func (s *Store) ListPublicActivity(ctx context.Context, limit int) ([]Op, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT o.id, o.space_id, COALESCE(o.node_id, ''), COALESCE(n.title, ''),
		       o.actor, o.actor_id, COALESCE(u.kind, 'human'), o.op, o.payload, o.created_at
		FROM ops o
		JOIN spaces s ON s.id = o.space_id AND s.visibility = 'public'
		LEFT JOIN users u ON u.id = o.actor_id
		LEFT JOIN nodes n ON n.id = o.node_id
		ORDER BY o.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ops []Op
	for rows.Next() {
		var o Op
		if err := rows.Scan(&o.ID, &o.SpaceID, &o.NodeID, &o.NodeTitle, &o.Actor, &o.ActorID, &o.ActorKind, &o.Op, &o.Payload, &o.CreatedAt); err != nil {
			return nil, err
		}
		ops = append(ops, o)
	}
	return ops, rows.Err()
}

// ListAvailableTasks returns open, unassigned tasks from public spaces.
// If query is non-empty, filters by title/body text search.
func (s *Store) ListAvailableTasks(ctx context.Context, query, priority string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 50
	}
	baseQuery := `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, n.verdict, n.rating, 0, 0, 0
		FROM nodes n
		JOIN spaces s ON s.id = n.space_id AND s.visibility = 'public'
		WHERE n.kind = 'task' AND n.state = 'open' AND n.assignee = ''
		  AND n.parent_id IS NULL`
	args := []any{}
	argN := 1
	if query != "" {
		baseQuery += fmt.Sprintf(" AND (n.title ILIKE '%%' || $%d || '%%' OR n.body ILIKE '%%' || $%d || '%%')", argN, argN)
		args = append(args, query)
		argN++
	}
	if priority != "" {
		baseQuery += fmt.Sprintf(" AND n.priority = $%d", argN)
		args = append(args, priority)
		argN++
	}
	baseQuery += fmt.Sprintf(" ORDER BY n.priority = 'urgent' DESC, n.priority = 'high' DESC, n.created_at DESC LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind,
			pq.Array(&n.Tags), &n.Pinned, &dueDate, &n.CreatedAt, &n.UpdatedAt,
			&n.Verdict, &n.Rating, &n.ChildCount, &n.ChildDone, &n.BlockerCount,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		tasks = append(tasks, n)
	}
	return tasks, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Invites
// ────────────────────────────────────────────────────────────────────

// CreateInvite generates an invite token for a space.
func (s *Store) CreateInvite(ctx context.Context, spaceID, createdBy string) (string, error) {
	token := newID()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO invites (token, space_id, created_by) VALUES ($1, $2, $3)`,
		token, spaceID, createdBy)
	if err != nil {
		return "", fmt.Errorf("create invite: %w", err)
	}
	return token, nil
}

// GetInviteSpaceID returns the space ID for an invite token, or empty if invalid.
func (s *Store) GetInviteSpaceID(ctx context.Context, token string) string {
	var spaceID string
	s.db.QueryRowContext(ctx, `SELECT space_id FROM invites WHERE token = $1`, token).Scan(&spaceID)
	return spaceID
}

// GetInviteToken returns an existing invite token for a space (if any), or empty.
func (s *Store) GetInviteToken(ctx context.Context, spaceID string) string {
	var token string
	s.db.QueryRowContext(ctx, `SELECT token FROM invites WHERE space_id = $1 ORDER BY created_at DESC LIMIT 1`, spaceID).Scan(&token)
	return token
}

// InviteCode holds full invite code data including expiry and usage limits.
type InviteCode struct {
	Token     string
	SpaceID   string
	CreatedBy string
	CreatedAt time.Time
	ExpiresAt *time.Time
	MaxUses   int
	UseCount  int
}

// CreateInviteCode creates an invite code with optional expiry and max uses (0 = unlimited).
func (s *Store) CreateInviteCode(ctx context.Context, spaceID, createdBy string, expiresAt *time.Time, maxUses int) (string, error) {
	token := newID()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO invites (token, space_id, created_by, expires_at, max_uses) VALUES ($1, $2, $3, $4, $5)`,
		token, spaceID, createdBy, expiresAt, maxUses)
	if err != nil {
		return "", fmt.Errorf("create invite code: %w", err)
	}
	return token, nil
}

// GetInviteCode returns an invite code if it exists and is still valid (not expired, not exhausted).
// Returns nil, nil if not found, expired, or exhausted.
func (s *Store) GetInviteCode(ctx context.Context, token string) (*InviteCode, error) {
	var inv InviteCode
	var expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT token, space_id, created_by, created_at, expires_at, max_uses, use_count
		 FROM invites WHERE token = $1`, token).
		Scan(&inv.Token, &inv.SpaceID, &inv.CreatedBy, &inv.CreatedAt, &expiresAt, &inv.MaxUses, &inv.UseCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invite code: %w", err)
	}
	if expiresAt.Valid {
		inv.ExpiresAt = &expiresAt.Time
		if time.Now().After(expiresAt.Time) {
			return nil, nil
		}
	}
	if inv.MaxUses > 0 && inv.UseCount >= inv.MaxUses {
		return nil, nil
	}
	return &inv, nil
}

// UseInviteCode records a use by userID. Idempotent per (token, userID) — increments use_count only on first use.
func (s *Store) UseInviteCode(ctx context.Context, token, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO invite_uses (token, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		token, userID)
	if err != nil {
		return fmt.Errorf("use invite code: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows > 0 {
		_, err = s.db.ExecContext(ctx,
			`UPDATE invites SET use_count = use_count + 1 WHERE token = $1`, token)
		if err != nil {
			return fmt.Errorf("increment invite use count: %w", err)
		}
	}
	return nil
}

// ListInvites returns all invite codes for a space, newest first.
func (s *Store) ListInvites(ctx context.Context, spaceID string) ([]InviteCode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT token, space_id, created_by, created_at, expires_at, max_uses, use_count
		 FROM invites WHERE space_id = $1 ORDER BY created_at DESC LIMIT 50`, spaceID)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()
	var codes []InviteCode
	for rows.Next() {
		var inv InviteCode
		var expiresAt sql.NullTime
		if err := rows.Scan(&inv.Token, &inv.SpaceID, &inv.CreatedBy, &inv.CreatedAt, &expiresAt, &inv.MaxUses, &inv.UseCount); err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		if expiresAt.Valid {
			inv.ExpiresAt = &expiresAt.Time
		}
		codes = append(codes, inv)
	}
	return codes, rows.Err()
}

// RevokeInvite deletes an invite code by token.
func (s *Store) RevokeInvite(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM invites WHERE token = $1`, token)
	return err
}

// ────────────────────────────────────────────────────────────────────
// User work history
// ────────────────────────────────────────────────────────────────────

// CompletedTask is a task completed by a user, with space context.
type CompletedTask struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	SpaceSlug string    `json:"space_slug"`
	SpaceName string    `json:"space_name"`
	DoneAt    time.Time `json:"done_at"`
}

// ListCompletedByUser returns tasks completed by a user (via complete op) in public spaces.
func (s *Store) ListCompletedByUser(ctx context.Context, userID string, limit int) ([]CompletedTask, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.title, s.slug, s.name, o.created_at
		FROM ops o
		JOIN nodes n ON n.id = o.node_id AND n.kind = 'task' AND n.state = 'done'
		JOIN spaces s ON s.id = o.space_id AND s.visibility = 'public'
		WHERE o.actor_id = $1 AND o.op = 'complete'
		ORDER BY o.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list completed by user: %w", err)
	}
	defer rows.Close()

	var tasks []CompletedTask
	for rows.Next() {
		var t CompletedTask
		if err := rows.Scan(&t.ID, &t.Title, &t.SpaceSlug, &t.SpaceName, &t.DoneAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Notifications
// ────────────────────────────────────────────────────────────────────

// Notification is a user-facing notification.
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	OpID      string    `json:"op_id"`
	SpaceID   string    `json:"space_id"`
	NodeID    string    `json:"node_id,omitempty"` // resolved from ops table
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
	// Resolved at query time:
	SpaceSlug string `json:"space_slug"`
	SpaceName string `json:"space_name"`
}

// CreateNotification records a notification for a user.
func (s *Store) CreateNotification(ctx context.Context, userID, opID, spaceID, message string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO notifications (id, user_id, op_id, space_id, message) VALUES ($1, $2, $3, $4, $5)`,
		newID(), userID, opID, spaceID, message)
	return err
}

// ListNotifications returns recent notifications for a user.
func (s *Store) ListNotifications(ctx context.Context, userID string, limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.user_id, n.op_id, n.space_id, COALESCE(o.node_id, ''), n.message, n.read, n.created_at,
		       COALESCE(s.slug, ''), COALESCE(s.name, '')
		FROM notifications n
		LEFT JOIN spaces s ON s.id = n.space_id
		LEFT JOIN ops o ON o.id = n.op_id
		WHERE n.user_id = $1
		ORDER BY n.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifs []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.OpID, &n.SpaceID, &n.NodeID, &n.Message, &n.Read, &n.CreatedAt,
			&n.SpaceSlug, &n.SpaceName); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

// UnreadCount returns the number of unread notifications for a user.
func (s *Store) UnreadCount(ctx context.Context, userID string) int {
	var count int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false`, userID).Scan(&count)
	return count
}

// MarkNotificationsRead marks all notifications for a user as read.
func (s *Store) MarkNotificationsRead(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`, userID)
	return err
}

// ────────────────────────────────────────────────────────────────────
// Pinning (Layer 12 — Culture)
// ────────────────────────────────────────────────────────────────────

// MarkConversationRead updates the read state for a user in a conversation.
func (s *Store) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO read_state (user_id, conversation_id, last_read_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (user_id, conversation_id) DO UPDATE SET last_read_at = NOW()`,
		userID, conversationID)
	return err
}

// EditNodeBody updates a node's body and marks it as edited.
func (s *Store) EditNodeBody(ctx context.Context, nodeID, newBody string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET body = $1, updated_at = NOW() WHERE id = $2`, newBody, nodeID)
	if err != nil {
		return fmt.Errorf("edit node: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateNodeCauses sets the causes list for a node.
// Used to retroactively populate causes on nodes that were created without them
// (Invariant 2: CAUSALITY — every event has declared causes).
func (s *Store) UpdateNodeCauses(ctx context.Context, nodeID string, causes []string) error {
	if causes == nil {
		causes = []string{}
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET causes = $1 WHERE id = $2`, pq.Array(causes), nodeID)
	if err != nil {
		return fmt.Errorf("update node causes: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SoftDeleteNode replaces the body with a tombstone marker.
func (s *Store) SoftDeleteNode(ctx context.Context, nodeID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET body = '[deleted]', state = 'deleted', updated_at = NOW() WHERE id = $1`, nodeID)
	if err != nil {
		return fmt.Errorf("soft delete node: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPinned sets the pinned status of a node.
func (s *Store) SetPinned(ctx context.Context, nodeID string, pinned bool) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET pinned = $1, updated_at = NOW() WHERE id = $2`, pinned, nodeID)
	if err != nil {
		return fmt.Errorf("set pinned: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListPinnedNodes returns pinned nodes in a space.
func (s *Store) ListPinnedNodes(ctx context.Context, spaceID string) ([]Node, error) {
	return s.ListNodes(ctx, ListNodesParams{SpaceID: spaceID, Pinned: true, Limit: 20})
}

// ────────────────────────────────────────────────────────────────────
// Changelog (Layer 5 — Build)
// ────────────────────────────────────────────────────────────────────

// ChangelogEntry is a completed task with its completion op.
type ChangelogEntry struct {
	Node
	CompletedBy   string    `json:"completed_by"`
	CompletedByKind string  `json:"completed_by_kind"`
	CompletedAt   time.Time `json:"completed_at"`
}

// ListChangelog returns recently completed tasks in a space, most recent first.
func (s *Store) ListChangelog(ctx context.Context, spaceID string, limit int) ([]ChangelogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, 0, 0, 0,
		       o.actor, COALESCE(u.kind, 'human'), o.created_at
		FROM nodes n
		JOIN ops o ON o.node_id = n.id AND o.op = 'complete'
		LEFT JOIN users u ON u.id = o.actor_id
		WHERE n.space_id = $1 AND n.kind = 'task' AND n.state = 'done'
		ORDER BY o.created_at DESC
		LIMIT $2`, spaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list changelog: %w", err)
	}
	defer rows.Close()

	var entries []ChangelogEntry
	for rows.Next() {
		var e ChangelogEntry
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&e.ID, &e.SpaceID, &parentID, &e.Kind, &e.Title, &e.Body,
			&e.State, &e.Priority, &e.Assignee, &e.AssigneeID, &e.Author, &e.AuthorID, &e.AuthorKind,
			pq.Array(&e.Tags), &e.Pinned, &dueDate, &e.CreatedAt, &e.UpdatedAt,
			&e.ChildCount, &e.ChildDone, &e.BlockerCount,
			&e.CompletedBy, &e.CompletedByKind, &e.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan changelog: %w", err)
		}
		if parentID.Valid {
			e.ParentID = parentID.String
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Governance (Layer 11)
// ────────────────────────────────────────────────────────────────────

// ProposalWithVotes is a proposal node with vote tallies and quorum state.
type ProposalWithVotes struct {
	Node
	VotesYes       int    `json:"votes_yes"`
	VotesNo        int    `json:"votes_no"`
	QuorumPct      int    `json:"quorum_pct"`       // 0 = no quorum enforcement
	VotingBody     string `json:"voting_body"`      // "all", "council", "team"
	EffectiveVotes int    `json:"effective_votes"`  // direct + delegated vote count
	EligibleCount  int    `json:"eligible_count"`   // eligible voter count for quorum display
}

// ListProposals returns proposals in a space with vote counts.
func (s *Store) ListProposals(ctx context.Context, spaceID, stateFilter string, limit int) ([]ProposalWithVotes, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, 0, 0, 0,
		       COALESCE((SELECT COUNT(*) FROM ops o WHERE o.node_id = n.id AND o.op = 'vote' AND o.payload->>'vote' = 'yes'), 0),
		       COALESCE((SELECT COUNT(*) FROM ops o WHERE o.node_id = n.id AND o.op = 'vote' AND o.payload->>'vote' = 'no'), 0),
		       n.quorum_pct, n.voting_body
		FROM nodes n
		WHERE n.space_id = $1 AND n.kind = 'proposal'`
	args := []any{spaceID}
	if stateFilter != "" {
		q += ` AND n.state = $3`
		args = append(args, limit, stateFilter)
		q += ` ORDER BY n.created_at DESC LIMIT $2`
	} else {
		args = append(args, limit)
		q += ` ORDER BY n.state = 'open' DESC, n.created_at DESC LIMIT $2`
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	defer rows.Close()

	var proposals []ProposalWithVotes
	for rows.Next() {
		var p ProposalWithVotes
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.SpaceID, &parentID, &p.Kind, &p.Title, &p.Body,
			&p.State, &p.Priority, &p.Assignee, &p.AssigneeID, &p.Author, &p.AuthorID, &p.AuthorKind,
			pq.Array(&p.Tags), &p.Pinned, &dueDate, &p.CreatedAt, &p.UpdatedAt,
			&p.ChildCount, &p.ChildDone, &p.BlockerCount,
			&p.VotesYes, &p.VotesNo,
			&p.QuorumPct, &p.VotingBody,
		); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		if parentID.Valid {
			p.ParentID = parentID.String
		}
		proposals = append(proposals, p)
	}
	return proposals, rows.Err()
}

// ListHiveActivity returns posts authored by agents, optionally filtered to a
// specific author by ID. limit must be > 0; if ≤ 0 it defaults to 20 (BOUNDED).
func (s *Store) ListHiveActivity(ctx context.Context, authorID string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 20
	}

	const cols = `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, n.verdict, n.rating
		FROM nodes n`

	var (
		rows *sql.Rows
		err  error
	)
	if authorID != "" {
		rows, err = s.db.QueryContext(ctx,
			cols+` WHERE n.kind = 'post' AND n.author_id = $1 ORDER BY n.created_at DESC LIMIT $2`,
			authorID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			cols+` WHERE n.kind = 'post' AND n.author_kind = 'agent' ORDER BY n.created_at DESC LIMIT $1`,
			limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list hive activity: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind,
			pq.Array(&n.Tags), &n.Pinned, &dueDate, &n.CreatedAt, &n.UpdatedAt, &n.Verdict, &n.Rating,
		); err != nil {
			return nil, fmt.Errorf("scan hive node: %w", err)
		}
		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ListHiveAgentTasks returns open tasks authored by agents, optionally scoped to
// a specific actorID. limit must be > 0; defaults to 10. BOUNDED.
func (s *Store) ListHiveAgentTasks(ctx context.Context, actorID string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 10
	}
	const cols = `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, n.verdict, n.rating
		FROM nodes n`
	var rows *sql.Rows
	var err error
	if actorID != "" {
		rows, err = s.db.QueryContext(ctx, cols+`
			WHERE n.kind = 'task' AND n.state = 'open' AND n.author_id = $1
			ORDER BY n.created_at DESC LIMIT $2`, actorID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, cols+`
			WHERE n.kind = 'task' AND n.state = 'open' AND n.author_kind = 'agent'
			ORDER BY n.created_at DESC LIMIT $1`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list hive agent tasks: %w", err)
	}
	defer rows.Close()
	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind,
			pq.Array(&n.Tags), &n.Pinned, &dueDate, &n.CreatedAt, &n.UpdatedAt, &n.Verdict, &n.Rating,
		); err != nil {
			return nil, fmt.Errorf("scan hive agent task: %w", err)
		}
		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// GetHiveCurrentTask returns the most recent open task authored by the given actor.
// If actorID is empty, falls back to matching any agent (author_kind = 'agent').
// BOUNDED: at most 1 row.
func (s *Store) GetHiveCurrentTask(ctx context.Context, actorID string) (*Node, error) {
	const cols = `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, n.verdict, n.rating
		FROM nodes n`
	var row *sql.Row
	if actorID != "" {
		row = s.db.QueryRowContext(ctx, cols+`
			WHERE n.kind = 'task' AND n.state = 'open' AND n.author_id = $1
			ORDER BY n.created_at DESC LIMIT 1`, actorID)
	} else {
		row = s.db.QueryRowContext(ctx, cols+`
			WHERE n.kind = 'task' AND n.state = 'open' AND n.author_kind = 'agent'
			ORDER BY n.created_at DESC LIMIT 1`)
	}
	var n Node
	var parentID sql.NullString
	var dueDate sql.NullTime
	err := row.Scan(
		&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
		&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind,
		pq.Array(&n.Tags), &n.Pinned, &dueDate, &n.CreatedAt, &n.UpdatedAt, &n.Verdict, &n.Rating,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get hive current task: %w", err)
	}
	if parentID.Valid {
		n.ParentID = parentID.String
	}
	if dueDate.Valid {
		d := dueDate.Time
		n.DueDate = &d
	}
	return &n, nil
}

// GetHiveTotals returns the count of ops and last active timestamp for the given actor.
// If actorID is empty, falls back to matching any agent (users.kind = 'agent'). BOUNDED.
func (s *Store) GetHiveTotals(ctx context.Context, actorID string) (totalOps int, lastActive time.Time, err error) {
	var t sql.NullTime
	if actorID != "" {
		err = s.db.QueryRowContext(ctx, `
			SELECT COUNT(id), MAX(created_at) FROM ops WHERE actor_id = $1`,
			actorID).Scan(&totalOps, &t)
	} else {
		err = s.db.QueryRowContext(ctx, `
			SELECT COUNT(o.id), MAX(o.created_at)
			FROM ops o
			JOIN users u ON u.id = o.actor_id
			WHERE u.kind = 'agent'`).Scan(&totalOps, &t)
	}
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("get hive totals: %w", err)
	}
	if t.Valid {
		lastActive = t.Time
	}
	return
}

// GetHiveAgentID returns the actor ID of the first agent user registered with
// an API key (users.kind = 'agent'). Returns "" if no agent has been linked
// to an API key. BOUNDED: LIMIT 1.
func (s *Store) GetHiveAgentID(ctx context.Context) string {
	var id string
	s.db.QueryRowContext(ctx, `
		SELECT u.id FROM users u
		JOIN api_keys k ON k.agent_id = u.id
		WHERE u.kind = 'agent'
		ORDER BY k.created_at ASC
		LIMIT 1`).Scan(&id)
	return id
}

// AppendHiveDiagnostic stores a phase event from the hive runner.
// id is caller-supplied (e.g. a UUID from the runner); ON CONFLICT DO NOTHING
// ensures duplicate POSTs are idempotent. payload is the full JSON event.
func (s *Store) AppendHiveDiagnostic(ctx context.Context, phase, outcome string, costUSD float64, payload []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO hive_diagnostics (id, phase, outcome, cost_usd, payload)
		 VALUES ($1, $2, $3, $4, $5)`,
		newID(), phase, outcome, costUSD, payload)
	return err
}

// ListHiveDiagnostics returns the most recent phase events, newest first.
// limit must be > 0; defaults to 10 (BOUNDED).
func (s *Store) ListHiveDiagnostics(ctx context.Context, limit int) ([]DiagEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT phase, outcome, cost_usd, created_at FROM hive_diagnostics
		 ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list hive diagnostics: %w", err)
	}
	defer rows.Close()
	var entries []DiagEntry
	for rows.Next() {
		var e DiagEntry
		if err := rows.Scan(&e.Phase, &e.Outcome, &e.CostUSD, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan hive diagnostic: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// HasVoted checks if a user has already voted on a proposal.
func (s *Store) HasVoted(ctx context.Context, nodeID, actorID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM ops WHERE node_id = $1 AND actor_id = $2 AND op = 'vote')`,
		nodeID, actorID).Scan(&exists)
	return exists
}

// ────────────────────────────────────────────────────────────────────
// Governance: delegation and quorum
// ────────────────────────────────────────────────────────────────────

// SetProposalConfig sets quorum_pct and voting_body on a proposal node.
func (s *Store) SetProposalConfig(ctx context.Context, nodeID string, quorumPct int, votingBody string) error {
	if votingBody == "" {
		votingBody = VotingBodyAll
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET quorum_pct = $1, voting_body = $2 WHERE id = $3`,
		quorumPct, votingBody, nodeID)
	return err
}

// Delegate records that delegatorID delegates their vote in spaceID to delegateID.
// Returns an error if delegatorID == delegateID or if a circular delegation would result.
func (s *Store) Delegate(ctx context.Context, spaceID, delegatorID, delegateID string) error {
	if delegatorID == delegateID {
		return fmt.Errorf("cannot delegate to yourself")
	}
	// Prevent circular delegation: check if delegateID has delegated to delegatorID.
	var targetDelegate string
	_ = s.db.QueryRowContext(ctx,
		`SELECT delegate_id FROM delegations WHERE space_id = $1 AND delegator_id = $2`,
		spaceID, delegateID).Scan(&targetDelegate)
	if targetDelegate == delegatorID {
		return fmt.Errorf("circular delegation: delegate has already delegated to you")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO delegations (space_id, delegator_id, delegate_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (space_id, delegator_id) DO UPDATE SET delegate_id = $3, created_at = NOW()`,
		spaceID, delegatorID, delegateID)
	return err
}

// Undelegate removes a delegation for delegatorID in spaceID.
func (s *Store) Undelegate(ctx context.Context, spaceID, delegatorID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM delegations WHERE space_id = $1 AND delegator_id = $2`,
		spaceID, delegatorID)
	return err
}

// HasDelegated returns true if delegatorID has an active delegation in spaceID.
func (s *Store) HasDelegated(ctx context.Context, spaceID, delegatorID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM delegations WHERE space_id = $1 AND delegator_id = $2)`,
		spaceID, delegatorID).Scan(&exists)
	return exists
}

// GetSpaceMemberCount returns the number of distinct members in a space
// (space_members rows + the owner if not in space_members).
func (s *Store) GetSpaceMemberCount(ctx context.Context, spaceID string) int {
	var count int
	s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT user_id FROM space_members WHERE space_id = $1
			UNION
			SELECT owner_id FROM spaces WHERE id = $1
		) uniq`, spaceID).Scan(&count)
	return count
}

// GetVotingBodyMemberCount returns the eligible voter count for a proposal based on its voting_body.
// VotingBodyAll → all space members (same as GetSpaceMemberCount).
// VotingBodyCouncil → distinct members across all KindCouncil nodes in the space.
// VotingBodyTeam → distinct members across all KindTeam nodes in the space.
func (s *Store) GetVotingBodyMemberCount(ctx context.Context, spaceID, votingBody string) int {
	switch votingBody {
	case VotingBodyCouncil, VotingBodyTeam:
		var count int
		s.db.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT nm.user_id)
			FROM node_members nm
			JOIN nodes n ON n.id = nm.node_id
			WHERE n.space_id = $1 AND n.kind = $2`,
			spaceID, votingBody).Scan(&count)
		return count
	default:
		return s.GetSpaceMemberCount(ctx, spaceID)
	}
}

// GetEffectiveVoteCount returns the number of unique voters (direct + delegated)
// who have voted on the given proposal node in the given space.
func (s *Store) GetEffectiveVoteCount(ctx context.Context, spaceID, nodeID string) int {
	var count int
	s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT DISTINCT o.actor_id FROM ops o
			WHERE o.node_id = $1 AND o.op = 'vote'
			UNION
			SELECT DISTINCT d.delegator_id FROM delegations d
			JOIN ops o ON o.actor_id = d.delegate_id
			WHERE o.node_id = $1 AND o.op = 'vote' AND d.space_id = $2
		) uniq`, nodeID, spaceID).Scan(&count)
	return count
}

// CheckAndAutoCloseProposal closes a proposal if quorum has been reached.
// If yes_count > no_count the outcome is "passed"; otherwise "rejected".
// Returns true if the proposal was auto-closed.
func (s *Store) CheckAndAutoCloseProposal(ctx context.Context, spaceID, nodeID string) (bool, error) {
	var quorumPct int
	var state, votingBody string
	err := s.db.QueryRowContext(ctx,
		`SELECT quorum_pct, state, voting_body FROM nodes WHERE id = $1`, nodeID).Scan(&quorumPct, &state, &votingBody)
	if err != nil || state != ProposalOpen || quorumPct == 0 {
		return false, err
	}

	eligible := s.GetVotingBodyMemberCount(ctx, spaceID, votingBody)
	if eligible == 0 {
		return false, nil
	}
	effective := s.GetEffectiveVoteCount(ctx, spaceID, nodeID)
	if effective*100 < quorumPct*eligible {
		return false, nil
	}

	// Quorum met — determine outcome.
	var yes, no int
	s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN payload->>'vote' = 'yes' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN payload->>'vote' = 'no' THEN 1 ELSE 0 END), 0)
		FROM ops WHERE node_id = $1 AND op = 'vote'`, nodeID).Scan(&yes, &no)

	newState := ProposalFailed
	outcome := "rejected"
	if yes > no {
		newState = ProposalPassed
		outcome = "passed"
	}
	if err := s.UpdateNodeState(ctx, nodeID, newState); err != nil {
		return false, err
	}
	payload, _ := json.Marshal(map[string]string{"outcome": outcome, "trigger": "quorum"})
	s.RecordOp(ctx, spaceID, nodeID, "system", "", "close_proposal", payload)
	return true, nil
}

// DelegationRow represents a single delegation entry in a space, with names resolved.
type DelegationRow struct {
	DelegatorID   string
	DelegatorName string
	DelegateID    string
	DelegateName  string
}

// GetUserDelegation returns who actorID has delegated to in spaceID, if any.
func (s *Store) GetUserDelegation(ctx context.Context, spaceID, actorID string) (delegateID, delegateName string, exists bool) {
	var dID string
	if err := s.db.QueryRowContext(ctx,
		`SELECT delegate_id FROM delegations WHERE space_id = $1 AND delegator_id = $2`,
		spaceID, actorID).Scan(&dID); err != nil {
		return "", "", false
	}
	s.db.QueryRowContext(ctx,
		`SELECT user_name FROM space_members WHERE space_id = $1 AND user_id = $2`,
		spaceID, dID).Scan(&delegateName)
	if delegateName == "" {
		delegateName = dID
	}
	return dID, delegateName, true
}

// ListDelegations returns all delegations in a space with display names resolved (BOUNDED at limit).
func (s *Store) ListDelegations(ctx context.Context, spaceID string, limit int) ([]DelegationRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.delegator_id, COALESCE(sm1.user_name, d.delegator_id),
		       d.delegate_id,  COALESCE(sm2.user_name, d.delegate_id)
		FROM delegations d
		LEFT JOIN space_members sm1 ON sm1.space_id = d.space_id AND sm1.user_id = d.delegator_id
		LEFT JOIN space_members sm2 ON sm2.space_id = d.space_id AND sm2.user_id = d.delegate_id
		WHERE d.space_id = $1
		ORDER BY d.created_at DESC
		LIMIT $2`, spaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list delegations: %w", err)
	}
	defer rows.Close()
	var result []DelegationRow
	for rows.Next() {
		var r DelegationRow
		if err := rows.Scan(&r.DelegatorID, &r.DelegatorName, &r.DelegateID, &r.DelegateName); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Search
// ────────────────────────────────────────────────────────────────────

// SearchResults holds grouped results from a platform search.
type SearchResults struct {
	Spaces []Space
	Nodes  []DashboardTask // reuse DashboardTask for node + space context
	Users  []struct {
		Name string
		Kind string
	}
}

// Search performs a text search across public spaces, nodes, and users.
func (s *Store) Search(ctx context.Context, query string, limit int) SearchResults {
	if limit <= 0 {
		limit = 10
	}
	var results SearchResults
	if query == "" {
		return results
	}
	pattern := "%" + query + "%"

	// Spaces
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, slug, name, description, owner_id, kind, visibility, COALESCE(parent_id, ''), created_at
		 FROM spaces WHERE visibility = 'public' AND (name ILIKE $1 OR description ILIKE $1)
		 ORDER BY created_at DESC LIMIT $2`, pattern, limit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sp Space
			if rows.Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.ParentID, &sp.CreatedAt) == nil {
				results.Spaces = append(results.Spaces, sp)
			}
		}
	}

	// Nodes from public spaces
	rows2, err := s.db.QueryContext(ctx,
		`SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		        n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		        n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, 0, 0, 0, s.slug, s.name
		 FROM nodes n
		 JOIN spaces s ON s.id = n.space_id AND s.visibility = 'public'
		 WHERE n.title ILIKE $1 OR n.body ILIKE $1
		 ORDER BY n.created_at DESC LIMIT $2`, pattern, limit)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var dt DashboardTask
			var parentID sql.NullString
			var dueDate sql.NullTime
			if rows2.Scan(
				&dt.ID, &dt.SpaceID, &parentID, &dt.Kind, &dt.Title, &dt.Body,
				&dt.State, &dt.Priority, &dt.Assignee, &dt.AssigneeID, &dt.Author, &dt.AuthorID, &dt.AuthorKind,
				pq.Array(&dt.Tags), &dt.Pinned, &dueDate, &dt.CreatedAt, &dt.UpdatedAt,
				&dt.ChildCount, &dt.ChildDone, &dt.BlockerCount,
				&dt.SpaceSlug, &dt.SpaceName,
			) == nil {
				if parentID.Valid {
					dt.ParentID = parentID.String
				}
				results.Nodes = append(results.Nodes, dt)
			}
		}
	}

	// Users
	rows3, err := s.db.QueryContext(ctx,
		`SELECT name, kind FROM users WHERE name ILIKE $1 ORDER BY name LIMIT $2`, pattern, limit)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var u struct {
				Name string
				Kind string
			}
			if rows3.Scan(&u.Name, &u.Kind) == nil {
				results.Users = append(results.Users, u)
			}
		}
	}

	return results
}

// MessageSearchResult is a message matching a search query, with conversation context.
type MessageSearchResult struct {
	ID         string
	Body       string
	Author     string
	AuthorID   string
	AuthorKind string
	CreatedAt  time.Time
	ConvoID    string
	ConvoTitle string
}

// SearchMessages searches message bodies within a space's conversations.
// Optional fromAuthor filters by author name (ILIKE).
func (s *Store) SearchMessages(ctx context.Context, spaceID, query, fromAuthor string, limit int) ([]MessageSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if query == "" && fromAuthor == "" {
		return nil, nil
	}

	q := `SELECT m.id, m.body, m.author, m.author_id, m.author_kind, m.created_at,
	             c.id, c.title
	      FROM nodes m
	      JOIN nodes c ON c.id = m.parent_id AND c.kind = 'conversation'
	      WHERE m.space_id = $1 AND m.body != '[deleted]'`

	args := []any{spaceID}
	argN := 2

	if query != "" {
		q += fmt.Sprintf(" AND m.body ILIKE '%%' || $%d || '%%'", argN)
		args = append(args, query)
		argN++
	}
	if fromAuthor != "" {
		q += fmt.Sprintf(" AND m.author ILIKE '%%' || $%d || '%%'", argN)
		args = append(args, fromAuthor)
		argN++
	}

	q += fmt.Sprintf(" ORDER BY m.created_at DESC LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()

	var results []MessageSearchResult
	for rows.Next() {
		var r MessageSearchResult
		if err := rows.Scan(&r.ID, &r.Body, &r.Author, &r.AuthorID, &r.AuthorKind, &r.CreatedAt, &r.ConvoID, &r.ConvoTitle); err != nil {
			return nil, fmt.Errorf("scan message search: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Endorsements (Layer 9 — Relationship)
// ────────────────────────────────────────────────────────────────────

// Endorse records an endorsement from one user to another.
func (s *Store) Endorse(ctx context.Context, fromID, toID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO endorsements (from_id, to_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		fromID, toID)
	return err
}

// Unendorse removes an endorsement.
func (s *Store) Unendorse(ctx context.Context, fromID, toID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM endorsements WHERE from_id = $1 AND to_id = $2`,
		fromID, toID)
	return err
}

// CountEndorsements returns how many users have endorsed the given user.
func (s *Store) CountEndorsements(ctx context.Context, userID string) int {
	var count int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM endorsements WHERE to_id = $1`, userID).Scan(&count)
	return count
}

// HasEndorsed checks if fromID has endorsed toID.
func (s *Store) HasEndorsed(ctx context.Context, fromID, toID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM endorsements WHERE from_id = $1 AND to_id = $2)`,
		fromID, toID).Scan(&exists)
	return exists
}

// ListEndorsers returns the names of users who endorsed the given user.
func (s *Store) ListEndorsers(ctx context.Context, userID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.name FROM endorsements e JOIN users u ON u.id = e.from_id
		 WHERE e.to_id = $1 ORDER BY e.created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			names = append(names, name)
		}
	}
	return names, rows.Err()
}

// GetBulkEndorsementCounts returns endorsement counts for multiple targets (users or nodes).
func (s *Store) GetBulkEndorsementCounts(ctx context.Context, targetIDs []string) map[string]int {
	result := make(map[string]int)
	if len(targetIDs) == 0 {
		return result
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT to_id, COUNT(*) FROM endorsements WHERE to_id = ANY($1) GROUP BY to_id`,
		pq.Array(targetIDs))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var count int
		if rows.Scan(&id, &count) == nil {
			result[id] = count
		}
	}
	return result
}

// GetBulkUserEndorsements returns which targets the user has endorsed.
func (s *Store) GetBulkUserEndorsements(ctx context.Context, userID string, targetIDs []string) map[string]bool {
	result := make(map[string]bool)
	if len(targetIDs) == 0 || userID == "" {
		return result
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT to_id FROM endorsements WHERE from_id = $1 AND to_id = ANY($2)`,
		userID, pq.Array(targetIDs))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			result[id] = true
		}
	}
	return result
}

// ────────────────────────────────────────────────────────────────────
// Follows (Layer 3 — Social)
// ────────────────────────────────────────────────────────────────────

// Follow records that followerID follows followedID.
func (s *Store) Follow(ctx context.Context, followerID, followedID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO follows (follower_id, followed_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		followerID, followedID)
	return err
}

// Unfollow removes a follow relationship.
func (s *Store) Unfollow(ctx context.Context, followerID, followedID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM follows WHERE follower_id = $1 AND followed_id = $2`,
		followerID, followedID)
	return err
}

// IsFollowing checks if followerID follows followedID.
func (s *Store) IsFollowing(ctx context.Context, followerID, followedID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND followed_id = $2)`,
		followerID, followedID).Scan(&exists)
	return exists
}

// CountFollowers returns how many users follow the given user.
func (s *Store) CountFollowers(ctx context.Context, userID string) int {
	var count int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE followed_id = $1`, userID).Scan(&count)
	return count
}

// CountFollowing returns how many users the given user follows.
func (s *Store) CountFollowing(ctx context.Context, userID string) int {
	var count int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = $1`, userID).Scan(&count)
	return count
}

// ListFollowedIDs returns the IDs of users that userID follows.
func (s *Store) ListFollowedIDs(ctx context.Context, userID string) []string {
	if userID == "" {
		return nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT followed_id FROM follows WHERE follower_id = $1`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// ListRepostedNodeIDs returns node IDs reposted by any of the given user IDs, limited.
func (s *Store) ListRepostedNodeIDs(ctx context.Context, userIDs []string, limit int) []string {
	if len(userIDs) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT node_id FROM reposts WHERE user_id = ANY($1) ORDER BY node_id LIMIT $2`,
		pq.Array(userIDs), limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetRepostAttribution returns a map of nodeID → reposter user ID, for nodes
// reposted by any of the given user IDs. Returns one reposter per node (most recent).
func (s *Store) GetRepostAttribution(ctx context.Context, userIDs, nodeIDs []string) map[string]string {
	result := make(map[string]string)
	if len(userIDs) == 0 || len(nodeIDs) == 0 {
		return result
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT ON (node_id) node_id, user_id FROM reposts
		 WHERE user_id = ANY($1) AND node_id = ANY($2)
		 ORDER BY node_id, created_at DESC`,
		pq.Array(userIDs), pq.Array(nodeIDs))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var nodeID, userID string
		if rows.Scan(&nodeID, &userID) == nil {
			result[nodeID] = userID
		}
	}
	return result
}

// ────────────────────────────────────────────────────────────────────
// Reposts (Layer 3 — Social / Propagate)
// ────────────────────────────────────────────────────────────────────

// Repost records that userID reposted nodeID.
func (s *Store) Repost(ctx context.Context, userID, nodeID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reposts (user_id, node_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, nodeID)
	return err
}

// Unrepost removes a repost.
func (s *Store) Unrepost(ctx context.Context, userID, nodeID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM reposts WHERE user_id = $1 AND node_id = $2`,
		userID, nodeID)
	return err
}

// HasReposted checks if userID has reposted nodeID.
func (s *Store) HasReposted(ctx context.Context, userID, nodeID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM reposts WHERE user_id = $1 AND node_id = $2)`,
		userID, nodeID).Scan(&exists)
	return exists
}

// GetBulkRepostCounts returns repost counts for multiple nodes.
func (s *Store) GetBulkRepostCounts(ctx context.Context, nodeIDs []string) map[string]int {
	result := make(map[string]int)
	if len(nodeIDs) == 0 {
		return result
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT node_id, COUNT(*) FROM reposts WHERE node_id = ANY($1) GROUP BY node_id`,
		pq.Array(nodeIDs))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var count int
		if rows.Scan(&id, &count) == nil {
			result[id] = count
		}
	}
	return result
}

// GetBulkUserReposts returns which nodes the user has reposted.
func (s *Store) GetBulkUserReposts(ctx context.Context, userID string, nodeIDs []string) map[string]bool {
	result := make(map[string]bool)
	if len(nodeIDs) == 0 || userID == "" {
		return result
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT node_id FROM reposts WHERE user_id = $1 AND node_id = ANY($2)`,
		userID, pq.Array(nodeIDs))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			result[id] = true
		}
	}
	return result
}

// ────────────────────────────────────────────────────────────────────
// Reactions
// ────────────────────────────────────────────────────────────────────

// ToggleReaction adds or removes a reaction. Returns true if added, false if removed.
func (s *Store) ToggleReaction(ctx context.Context, nodeID, userID, emoji string) (bool, error) {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM reactions WHERE node_id=$1 AND user_id=$2 AND emoji=$3)`,
		nodeID, userID, emoji).Scan(&exists)
	if exists {
		_, err := s.db.ExecContext(ctx,
			`DELETE FROM reactions WHERE node_id=$1 AND user_id=$2 AND emoji=$3`,
			nodeID, userID, emoji)
		return false, err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reactions (node_id, user_id, emoji) VALUES ($1, $2, $3)`,
		nodeID, userID, emoji)
	return true, err
}

// GetNodeReactions returns aggregated reactions for a single node.
func (s *Store) GetNodeReactions(ctx context.Context, nodeID string) []Reaction {
	rows, err := s.db.QueryContext(ctx,
		`SELECT emoji, COUNT(*) as cnt, ARRAY_AGG(user_id) as users
		 FROM reactions WHERE node_id = $1
		 GROUP BY emoji ORDER BY MIN(created_at)`, nodeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var reactions []Reaction
	for rows.Next() {
		var r Reaction
		var users []string
		if err := rows.Scan(&r.Emoji, &r.Count, pq.Array(&users)); err == nil {
			r.Users = users
			reactions = append(reactions, r)
		}
	}
	return reactions
}

// GetBulkReactions returns reactions for multiple nodes. Returns map[nodeID][]Reaction.
func (s *Store) GetBulkReactions(ctx context.Context, nodeIDs []string) map[string][]Reaction {
	result := make(map[string][]Reaction)
	if len(nodeIDs) == 0 {
		return result
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT node_id, emoji, COUNT(*) as cnt, ARRAY_AGG(user_id) as users
		 FROM reactions WHERE node_id = ANY($1)
		 GROUP BY node_id, emoji ORDER BY node_id, MIN(created_at)`, pq.Array(nodeIDs))
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var nodeID string
		var r Reaction
		var users []string
		if err := rows.Scan(&nodeID, &r.Emoji, &r.Count, pq.Array(&users)); err == nil {
			r.Users = users
			result[nodeID] = append(result[nodeID], r)
		}
	}
	return result
}

// ────────────────────────────────────────────────────────────────────
// Reports
// ────────────────────────────────────────────────────────────────────

// Report is a report op with the associated node info.
type Report struct {
	Op
	NodeTitle string `json:"node_title"`
	NodeKind  string `json:"node_kind"`
	Reason    string `json:"reason"`
}

// ListReports returns unresolved report ops for a space. A report is "unresolved"
// if no resolve op exists for the same node_id.
func (s *Store) ListReports(ctx context.Context, spaceID string) ([]Report, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT o.id, o.space_id, COALESCE(o.node_id, ''), o.actor, o.actor_id,
		       COALESCE(u.kind, 'human'), o.op, o.payload, o.created_at,
		       COALESCE(n.title, ''), COALESCE(n.kind, '')
		FROM ops o
		LEFT JOIN users u ON u.id = o.actor_id
		LEFT JOIN nodes n ON n.id = o.node_id
		WHERE o.space_id = $1 AND o.op = 'report'
		  AND NOT EXISTS (
		      SELECT 1 FROM ops r WHERE r.space_id = $1 AND r.op = 'resolve' AND r.node_id = o.node_id
		  )
		ORDER BY o.created_at DESC
		LIMIT 50`, spaceID)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var r Report
		if err := rows.Scan(
			&r.ID, &r.SpaceID, &r.NodeID, &r.Actor, &r.ActorID,
			&r.ActorKind, &r.Op, &r.Payload, &r.CreatedAt,
			&r.NodeTitle, &r.NodeKind,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		// Extract reason from payload.
		var payload map[string]string
		if json.Unmarshal(r.Payload, &payload) == nil {
			r.Reason = payload["reason"]
		}
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Dashboard (cross-space queries)
// ────────────────────────────────────────────────────────────────────

// DashboardTask is a task with its space context.
type DashboardTask struct {
	Node
	SpaceSlug string `json:"space_slug"`
	SpaceName string `json:"space_name"`
}

// DashboardConversation is a conversation summary with space context.
type DashboardConversation struct {
	ConversationSummary
	SpaceSlug string `json:"space_slug"`
	SpaceName string `json:"space_name"`
}

// DashboardOp is an op with space context.
type DashboardOp struct {
	Op
	SpaceSlug string `json:"space_slug"`
	SpaceName string `json:"space_name"`
}

// ListUserTasks returns tasks where the user is the author or assignee, across all spaces.
// stateFilter: "" for open (not done/closed), "all" for everything, or a specific state like "done".
func (s *Store) ListUserTasks(ctx context.Context, userID string, stateFilter string, limit int) ([]DashboardTask, error) {
	if limit <= 0 {
		limit = 30
	}
	q := `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM node_deps d JOIN nodes b ON b.id = d.depends_on WHERE d.node_id = n.id AND b.state != 'done'), 0),
		       s.slug, s.name
		FROM nodes n
		JOIN spaces s ON s.id = n.space_id
		WHERE n.kind = 'task'
		  AND (n.author_id = $1 OR n.assignee_id = $1)`
	switch stateFilter {
	case "all":
		// no state filter
	case "done":
		q += ` AND n.state = 'done'`
	case "active":
		q += ` AND n.state = 'active'`
	case "review":
		q += ` AND n.state = 'review'`
	default:
		q += ` AND n.state NOT IN ('done', 'closed')`
	}
	q += ` ORDER BY n.priority = 'urgent' DESC, n.priority = 'high' DESC, n.updated_at DESC LIMIT $2`
	rows, err := s.db.QueryContext(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list user tasks: %w", err)
	}
	defer rows.Close()

	var tasks []DashboardTask
	for rows.Next() {
		var dt DashboardTask
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&dt.ID, &dt.SpaceID, &parentID, &dt.Kind, &dt.Title, &dt.Body,
			&dt.State, &dt.Priority, &dt.Assignee, &dt.AssigneeID, &dt.Author, &dt.AuthorID, &dt.AuthorKind,
			pq.Array(&dt.Tags), &dt.Pinned, &dueDate, &dt.CreatedAt, &dt.UpdatedAt,
			&dt.ChildCount, &dt.ChildDone, &dt.BlockerCount,
			&dt.SpaceSlug, &dt.SpaceName,
		); err != nil {
			return nil, fmt.Errorf("scan user task: %w", err)
		}
		if parentID.Valid {
			dt.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			dt.DueDate = &d
		}
		tasks = append(tasks, dt)
	}
	return tasks, rows.Err()
}

// ListUserConversations returns conversations a user participates in across all spaces.
func (s *Store) ListUserConversations(ctx context.Context, userID string, limit int) ([]DashboardConversation, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind, n.tags, n.pinned, n.due_date,
		       n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       0, 0,
		       lm.author, lm.author_kind, lm.body,
		       s.slug, s.name
		FROM nodes n
		LEFT JOIN LATERAL (
		    SELECT c.author, c.author_kind, c.body
		    FROM nodes c WHERE c.parent_id = n.id
		    ORDER BY c.created_at DESC LIMIT 1
		) lm ON true
		JOIN spaces s ON s.id = n.space_id
		WHERE n.kind = 'conversation'
		  AND ($1 = ANY(n.tags) OR n.author_id = $1)
		ORDER BY GREATEST(n.updated_at, COALESCE(
		    (SELECT MAX(c.created_at) FROM nodes c WHERE c.parent_id = n.id), n.created_at
		)) DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list user conversations: %w", err)
	}
	defer rows.Close()

	var convos []DashboardConversation
	for rows.Next() {
		var dc DashboardConversation
		var parentID sql.NullString
		var dueDate sql.NullTime
		var lastAuthor, lastAuthorKind, lastBody sql.NullString
		if err := rows.Scan(
			&dc.ID, &dc.SpaceID, &parentID, &dc.Kind, &dc.Title, &dc.Body,
			&dc.State, &dc.Priority, &dc.Assignee, &dc.AssigneeID, &dc.Author, &dc.AuthorID, &dc.AuthorKind, pq.Array(&dc.Tags), &dc.Pinned, &dueDate,
			&dc.CreatedAt, &dc.UpdatedAt,
			&dc.ChildCount, &dc.ChildDone, &dc.BlockerCount,
			&lastAuthor, &lastAuthorKind, &lastBody,
			&dc.SpaceSlug, &dc.SpaceName,
		); err != nil {
			return nil, fmt.Errorf("scan user conversation: %w", err)
		}
		if parentID.Valid {
			dc.ParentID = parentID.String
		}
		if lastAuthor.Valid {
			dc.LastAuthor = lastAuthor.String
		}
		if lastAuthorKind.Valid {
			dc.LastAuthorKind = lastAuthorKind.String
		}
		if lastBody.Valid {
			dc.LastBody = lastBody.String
		}
		dc.SpaceSlug = dc.SpaceSlug
		dc.SpaceName = dc.SpaceName
		convos = append(convos, dc)
	}
	return convos, rows.Err()
}

// ListUserAgentActivity returns recent agent actions in spaces the user owns or is a member of.
func (s *Store) ListUserAgentActivity(ctx context.Context, userID string, limit int) ([]DashboardOp, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT o.id, o.space_id, COALESCE(o.node_id, ''), COALESCE(n.title, ''),
		       o.actor, o.actor_id, COALESCE(u.kind, 'human'), o.op, o.payload, o.created_at,
		       s.slug, s.name
		FROM ops o
		JOIN users u ON u.id = o.actor_id AND u.kind = 'agent'
		JOIN spaces s ON s.id = o.space_id
		LEFT JOIN nodes n ON n.id = o.node_id
		WHERE s.owner_id = $1
		   OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id = s.id AND sm.user_id = $1)
		ORDER BY o.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list user agent activity: %w", err)
	}
	defer rows.Close()

	var ops []DashboardOp
	for rows.Next() {
		var do DashboardOp
		if err := rows.Scan(
			&do.ID, &do.SpaceID, &do.NodeID, &do.NodeTitle, &do.Actor, &do.ActorID,
			&do.ActorKind, &do.Op, &do.Payload, &do.CreatedAt,
			&do.SpaceSlug, &do.SpaceName,
		); err != nil {
			return nil, fmt.Errorf("scan agent activity: %w", err)
		}
		ops = append(ops, do)
	}
	return ops, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Dependencies
// ────────────────────────────────────────────────────────────────────

// AddDependency marks nodeID as depending on dependsOn.
func (s *Store) AddDependency(ctx context.Context, nodeID, dependsOn string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO node_deps (node_id, depends_on) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		nodeID, dependsOn)
	return err
}

// ListBlockers returns nodes that block the given node (incomplete dependencies).
func (s *Store) ListBlockers(ctx context.Context, nodeID string) ([]Node, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, 0, 0, 0
		FROM nodes n
		JOIN node_deps d ON d.depends_on = n.id
		WHERE d.node_id = $1 AND n.state != 'done'
		ORDER BY n.created_at`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind,
			pq.Array(&n.Tags), &n.Pinned, &dueDate, &n.CreatedAt, &n.UpdatedAt,
			&n.ChildCount, &n.ChildDone, &n.BlockerCount,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// RemoveDependency removes the dependency relationship between nodeID and dependsOn.
func (s *Store) RemoveDependency(ctx context.Context, nodeID, dependsOn string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM node_deps WHERE node_id = $1 AND depends_on = $2`,
		nodeID, dependsOn)
	return err
}

// ListDependencies returns all nodes that nodeID depends on (both done and not done).
func (s *Store) ListDependencies(ctx context.Context, nodeID string) ([]Node, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, 0, 0, 0
		FROM nodes n
		JOIN node_deps d ON d.depends_on = n.id
		WHERE d.node_id = $1
		ORDER BY n.state != 'done', n.created_at`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind,
			pq.Array(&n.Tags), &n.Pinned, &dueDate, &n.CreatedAt, &n.UpdatedAt,
			&n.ChildCount, &n.ChildDone, &n.BlockerCount,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ListDependents returns all nodes that depend on nodeID.
func (s *Store) ListDependents(ctx context.Context, nodeID string) ([]Node, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, COALESCE(n.parent_id, ''), n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind,
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, 0, 0, 0
		FROM nodes n
		JOIN node_deps d ON d.node_id = n.id
		WHERE d.depends_on = $1
		ORDER BY n.state != 'done', n.created_at`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []Node
	for rows.Next() {
		var n Node
		var parentID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
			&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind,
			pq.Array(&n.Tags), &n.Pinned, &dueDate, &n.CreatedAt, &n.UpdatedAt,
			&n.ChildCount, &n.ChildDone, &n.BlockerCount,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			n.ParentID = parentID.String
		}
		if dueDate.Valid {
			d := dueDate.Time
			n.DueDate = &d
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Mind State
// ────────────────────────────────────────────────────────────────────

// SetMindState upserts a key-value pair for the Mind's context.
func (s *Store) SetMindState(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO mind_state (key, value, updated_at) VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
		key, value)
	return err
}

// GetMindState returns the value for a mind state key, or empty string.
func (s *Store) GetMindState(ctx context.Context, key string) string {
	var value string
	s.db.QueryRowContext(ctx, `SELECT value FROM mind_state WHERE key = $1`, key).Scan(&value)
	return value
}

// ────────────────────────────────────────────────────────────────────
// Knowledge (Layer 6)
// ────────────────────────────────────────────────────────────────────

// KnowledgeClaim is a claim node with challenge count and space info for cross-space display.
type KnowledgeClaim struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	State      string    `json:"state"`
	Author     string    `json:"author"`
	AuthorID   string    `json:"author_id"`
	AuthorKind string    `json:"author_kind"`
	SpaceSlug  string    `json:"space_slug"`
	SpaceName  string    `json:"space_name"`
	Challenges int       `json:"challenges"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListKnowledgeClaims returns claims from public spaces, optionally filtered by state or query.
func (s *Store) ListKnowledgeClaims(ctx context.Context, stateFilter, query string, limit int) ([]KnowledgeClaim, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `
		SELECT n.id, n.title, n.body, n.state, n.author, n.author_id,
		       COALESCE(u.kind, 'human'), s.slug, s.name,
		       (SELECT COUNT(*) FROM ops o WHERE o.node_id = n.id AND o.op = 'challenge'),
		       n.created_at
		FROM nodes n
		JOIN spaces s ON s.id = n.space_id AND s.visibility = 'public'
		LEFT JOIN users u ON u.id = n.author_id
		WHERE n.kind = 'claim'`
	args := []any{}
	argN := 1
	if stateFilter != "" {
		q += fmt.Sprintf(" AND n.state = $%d", argN)
		args = append(args, stateFilter)
		argN++
	}
	if query != "" {
		q += fmt.Sprintf(" AND (n.title ILIKE '%%' || $%d || '%%' OR n.body ILIKE '%%' || $%d || '%%')", argN, argN)
		args = append(args, query)
		argN++
	}
	q += fmt.Sprintf(" ORDER BY n.created_at DESC LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var claims []KnowledgeClaim
	for rows.Next() {
		var c KnowledgeClaim
		if err := rows.Scan(&c.ID, &c.Title, &c.Body, &c.State, &c.Author, &c.AuthorID,
			&c.AuthorKind, &c.SpaceSlug, &c.SpaceName, &c.Challenges, &c.CreatedAt); err != nil {
			return nil, err
		}
		claims = append(claims, c)
	}
	return claims, rows.Err()
}

// MaxLessonNumber returns the highest "Lesson N" claim number in a space.
// Titles must match "^Lesson [0-9]+" (case-sensitive). Returns 0 if no
// numbered lessons exist. Server-side aggregate avoids client-side scan
// truncation as lesson count grows (Invariant 13: BOUNDED).
func (s *Store) MaxLessonNumber(ctx context.Context, spaceID string) (int, error) {
	var max int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(
			CASE WHEN title ~ '^Lesson [0-9]+'
			THEN CAST(REGEXP_REPLACE(title, '^Lesson ([0-9]+).*', '\1') AS INTEGER)
			ELSE 0 END
		), 0)
		FROM nodes
		WHERE space_id = $1 AND kind = 'claim' AND parent_id IS NULL
	`, spaceID).Scan(&max)
	return max, err
}

// CountChallenges returns the number of challenge ops for a given node.
func (s *Store) CountChallenges(ctx context.Context, nodeID string) int {
	var count int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ops WHERE node_id = $1 AND op = 'challenge'`, nodeID).Scan(&count)
	return count
}

// ────────────────────────────────────────────────────────────────────
// Agent Personas
// ────────────────────────────────────────────────────────────────────

// AgentPersona is a named agent persona seeded from agents/*.md files.
//
// Status reflects the role's maturity as declared in the source .md file's
// leading `<!-- Status: X -->` comment. Seven values: running, ready, designed,
// aspirational, challenged, absorbed, retired. Personas with status
// "absorbed" or "retired" are forced inactive during seeding regardless
// of the personaActive map — the source file is the authority.
type AgentPersona struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Display     string     `json:"display"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Prompt      string     `json:"prompt"`
	Model       string     `json:"model"`
	Active      bool       `json:"active"`
	Status      string     `json:"status"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	SessionID   string     `json:"session_id"`
}

// UpsertAgentPersona inserts or updates an agent persona by name.
func (s *Store) UpsertAgentPersona(ctx context.Context, p AgentPersona) error {
	status := p.Status
	if status == "" {
		status = "ready"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_personas (id, name, display, description, category, prompt, model, active, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (name) DO UPDATE SET
			display = EXCLUDED.display,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			prompt = EXCLUDED.prompt,
			model = EXCLUDED.model,
			active = EXCLUDED.active,
			status = EXCLUDED.status`,
		newID(), p.Name, p.Display, p.Description, p.Category, p.Prompt, p.Model, p.Active, status,
	)
	return err
}

// UpdateAgentPersonaLastSeen sets last_seen = NOW() for the named persona.
func (s *Store) UpdateAgentPersonaLastSeen(ctx context.Context, name string) {
	s.db.ExecContext(ctx, `UPDATE agent_personas SET last_seen = NOW() WHERE name = $1`, name)
}

// UpdateAgentSession sets the session_id for the named persona.
// Called by the hive pipeline after each successful Claude CLI call to persist
// the session UUID for warm resumption on subsequent iterations.
func (s *Store) UpdateAgentSession(ctx context.Context, name, sessionID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agent_personas SET session_id = $1 WHERE name = $2`,
		sessionID, name)
	return err
}

// GetAgentPersonaForConversation finds any agent participant in the given tag list
// (user IDs) and returns their persona, or nil if the conversation has no agent.
// Uses persona_name (the stable slug) for the join, not the mutable display name.
func (s *Store) GetAgentPersonaForConversation(ctx context.Context, tags []string) *AgentPersona {
	if len(tags) == 0 {
		return nil
	}
	var personaName string
	err := s.db.QueryRowContext(ctx,
		`SELECT persona_name FROM users WHERE id = ANY($1) AND kind = 'agent' AND persona_name IS NOT NULL LIMIT 1`,
		pq.Array(tags),
	).Scan(&personaName)
	if err != nil {
		return nil
	}
	return s.GetAgentPersona(ctx, personaName)
}

// GetAgentPersonasForConversations returns a map of conversation ID → AgentPersona
// for all conversations that have an agent participant. Uses a single query to
// avoid N+1 round-trips.
func (s *Store) GetAgentPersonasForConversations(ctx context.Context, convos []ConversationSummary) map[string]*AgentPersona {
	// Collect all unique user IDs across all conversations.
	seen := make(map[string]bool)
	for _, c := range convos {
		for _, id := range c.Tags {
			seen[id] = true
		}
	}
	if len(seen) == 0 {
		return map[string]*AgentPersona{}
	}
	allIDs := make([]string, 0, len(seen))
	for id := range seen {
		allIDs = append(allIDs, id)
	}

	// One query: find agent user IDs and their persona data.
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, ap.id, ap.name, ap.display, ap.description, ap.category, ap.prompt, ap.model, ap.active, ap.status, ap.last_seen
		FROM users u
		JOIN agent_personas ap ON ap.name = u.persona_name
		WHERE u.id = ANY($1) AND u.kind = 'agent' AND u.persona_name IS NOT NULL`,
		pq.Array(allIDs),
	)
	if err != nil {
		return map[string]*AgentPersona{}
	}
	defer rows.Close()

	// userID → persona
	byUserID := make(map[string]*AgentPersona)
	for rows.Next() {
		var userID string
		var p AgentPersona
		if err := rows.Scan(&userID, &p.ID, &p.Name, &p.Display, &p.Description, &p.Category, &p.Prompt, &p.Model, &p.Active, &p.Status, &p.LastSeen); err != nil {
			continue
		}
		byUserID[userID] = &p
	}

	// Map each conversation to the first agent persona found in its tags.
	result := make(map[string]*AgentPersona, len(convos))
	for _, c := range convos {
		for _, id := range c.Tags {
			if p, ok := byUserID[id]; ok {
				result[c.ID] = p
				break
			}
		}
	}
	return result
}

// GetAgentPersona returns a persona by slug name, or nil if not found.
func (s *Store) GetAgentPersona(ctx context.Context, name string) *AgentPersona {
	var p AgentPersona
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, display, description, category, prompt, model, active, status, last_seen
		 FROM agent_personas WHERE name = $1`, name,
	).Scan(&p.ID, &p.Name, &p.Display, &p.Description, &p.Category, &p.Prompt, &p.Model, &p.Active, &p.Status, &p.LastSeen)
	if err != nil {
		return nil
	}
	return &p
}

// ListAgentPersonas returns all active agent personas ordered by category, display.
func (s *Store) ListAgentPersonas(ctx context.Context) ([]AgentPersona, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, display, description, category, prompt, model, status, last_seen, session_id
		 FROM agent_personas WHERE active = true ORDER BY category, display`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var personas []AgentPersona
	for rows.Next() {
		var p AgentPersona
		var sessionID *string
		if err := rows.Scan(&p.ID, &p.Name, &p.Display, &p.Description, &p.Category, &p.Prompt, &p.Model, &p.Status, &p.LastSeen, &sessionID); err != nil {
			return nil, err
		}
		p.Active = true
		if sessionID != nil {
			p.SessionID = *sessionID
		}
		personas = append(personas, p)
	}
	return personas, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Agent Memories
// ────────────────────────────────────────────────────────────────────

// Memory is a stored memory record for an agent persona about a user.
type Memory struct {
	ID         string    `json:"id"`
	SpaceID    string    `json:"space_id"`
	UserID     string    `json:"user_id"`
	Persona    string    `json:"persona"`
	Content    string    `json:"content"`
	Kind       string    `json:"kind"`
	Importance int       `json:"importance"`
	CreatedAt  time.Time `json:"created_at"`
}

var validMemoryKinds = map[string]bool{
	"context":    true,
	"fact":       true,
	"preference": true,
}

// RememberForPersona stores a memory for a persona about a specific user.
// kind: "context" | "fact" | "preference"; importance: 1-10 (default 5).
func (s *Store) RememberForPersona(ctx context.Context, persona, userID, kind, content, sourceID string, importance int) error {
	if kind == "" {
		kind = "context"
	}
	if !validMemoryKinds[kind] {
		return fmt.Errorf("invalid memory kind %q: must be context, fact, or preference", kind)
	}
	if importance <= 0 || importance > 10 {
		importance = 5
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agent_memories (id, persona, user_id, memory, kind, source_id, importance)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		newID(), persona, userID, content, kind, sourceID, importance)
	return err
}

// RecallForPersona returns the most recent memories for a persona about a specific user,
// ordered by importance desc, then recency desc. Returns content strings for prompt injection.
func (s *Store) RecallForPersona(ctx context.Context, persona, userID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT memory FROM agent_memories WHERE persona = $1 AND user_id = $2
		 ORDER BY importance DESC, created_at DESC LIMIT $3`,
		persona, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memories []string
	for rows.Next() {
		var m string
		if rows.Scan(&m) == nil {
			memories = append(memories, m)
		}
	}
	return memories, rows.Err()
}

// memoryPersonaUser is the sentinel persona value used for user-level memories
// that are not tied to any specific agent persona.
const memoryPersonaUser = "__user__"

// RememberForUser stores a memory about a user that is not tied to any agent persona.
// kind: "context" | "fact" | "preference"; importance: 1-10 (default 5).
func (s *Store) RememberForUser(ctx context.Context, userID, kind, content, sourceID string, importance int) error {
	return s.RememberForPersona(ctx, memoryPersonaUser, userID, kind, content, sourceID, importance)
}

// RecallForUser returns user-level memories not tied to any agent persona,
// ordered by importance desc, then recency desc.
func (s *Store) RecallForUser(ctx context.Context, userID string, limit int) ([]string, error) {
	return s.RecallForPersona(ctx, memoryPersonaUser, userID, limit)
}

// RememberForUserInSpace stores a memory scoped to a specific space for a persona about a user.
// kind: "context" | "fact" | "preference"; importance: 1-10 (default 5).
func (s *Store) RememberForUserInSpace(ctx context.Context, spaceID, userID, persona, content, kind string, importance int) error {
	if kind == "" {
		kind = "context"
	}
	if !validMemoryKinds[kind] {
		return fmt.Errorf("invalid memory kind %q: must be context, fact, or preference", kind)
	}
	if importance <= 0 || importance > 10 {
		importance = 5
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agent_memories (id, space_id, persona, user_id, memory, kind, source_id, importance)
		 VALUES ($1, $2, $3, $4, $5, $6, '', $7)`,
		newID(), spaceID, persona, userID, content, kind, importance)
	return err
}

// RecallForUserInSpace returns memories for a persona about a user within a specific space,
// ordered by importance DESC, created_at DESC. Returns Memory structs.
func (s *Store) RecallForUserInSpace(ctx context.Context, spaceID, userID, persona string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, space_id, user_id, persona, memory, kind, importance, created_at
		 FROM agent_memories
		 WHERE space_id = $1 AND user_id = $2 AND persona = $3
		 ORDER BY importance DESC, created_at DESC LIMIT $4`,
		spaceID, userID, persona, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memories []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.SpaceID, &m.UserID, &m.Persona, &m.Content, &m.Kind, &m.Importance, &m.CreatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for _, s := range ss[1:] {
		out += sep + s
	}
	return out
}

// ────────────────────────────────────────────────────────────────────
// Demo space seed
// ────────────────────────────────────────────────────────────────────

const DemoSpaceSlug = "demo"

// SeedDemoSpace creates a public demo space with example content if it doesn't
// already exist. Idempotent — safe to call on every startup.
// Returns the demo space slug on success, "" on error.
func (s *Store) SeedDemoSpace(ctx context.Context) string {
	const demoGoogleID = "system:demo-agent"

	// Already exists — nothing to do.
	if existing, _ := s.GetSpaceBySlug(ctx, DemoSpaceSlug); existing != nil {
		return DemoSpaceSlug
	}

	// Upsert the demo agent user.
	var agentUserID string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (id, google_id, email, name, kind)
		VALUES ($1, $2, 'demo@agent.lovyou.ai', 'Demo Agent', 'agent')
		ON CONFLICT (google_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`,
		newID(), demoGoogleID,
	).Scan(&agentUserID)
	if err != nil {
		log.Printf("seed demo: upsert agent user: %v", err)
		return ""
	}

	// Create the demo space.
	space, err := s.CreateSpace(ctx,
		DemoSpaceSlug,
		"lovyou.ai Demo",
		"A live preview — tasks, posts, and agent conversations you can explore without signing in.",
		agentUserID,
		SpaceProject,
		VisibilityPublic,
	)
	if err != nil {
		log.Printf("seed demo: create space: %v", err)
		return ""
	}

	emptyPayload := json.RawMessage("{}")

	addTask := func(title, body, state, priority string) {
		n, err := s.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindTask,
			Title:      title,
			Body:       body,
			State:      state,
			Priority:   priority,
			Author:     "Demo Agent",
			AuthorID:   agentUserID,
			AuthorKind: "agent",
		})
		if err != nil || n == nil {
			return
		}
		op := "intend"
		switch state {
		case StateActive:
			op = "start"
		case StateDone:
			op = "complete"
		}
		s.RecordOp(ctx, space.ID, n.ID, "Demo Agent", agentUserID, op, emptyPayload) //nolint
	}

	// Board: tasks across all three visible states.
	addTask("Write product specification",
		"Define core user journeys, technical architecture, and success metrics for v1. Includes competitive analysis.",
		StateDone, PriorityHigh)
	addTask("Build authentication system",
		"Implement Google OAuth with session management and API key support for agents. Needs rate limiting and audit logging.",
		StateActive, PriorityHigh)
	addTask("Design the landing page",
		"Create a landing page that answers: what is this, why should I care, and how do I start in under 8 seconds.",
		StateOpen, PriorityMedium)
	addTask("Add full-text search across spaces",
		"Full-text search over nodes, spaces, and users — accessible from a command palette (⌘K).",
		StateOpen, PriorityLow)

	// Feed: an example post.
	post, err := s.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindPost,
		Title:      "Auth is live",
		Body:       "Google OAuth is working end-to-end. API keys work for agent access too. The auth layer is solid — moving on to the product next.",
		Author:     "Demo Agent",
		AuthorID:   agentUserID,
		AuthorKind: "agent",
	})
	if err == nil && post != nil {
		s.RecordOp(ctx, space.ID, post.ID, "Demo Agent", agentUserID, "express", emptyPayload) //nolint
	}

	// Chat: a conversation with an agent reply.
	conv, err := s.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindConversation,
		Title:      "What should we build next?",
		Body:       "What should we build next?",
		Author:     "Demo Agent",
		AuthorID:   agentUserID,
		AuthorKind: "agent",
	})
	if err == nil && conv != nil {
		s.RecordOp(ctx, space.ID, conv.ID, "Demo Agent", agentUserID, "converse", emptyPayload) //nolint

		reply, err := s.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindComment,
			ParentID:   conv.ID,
			Body:       "Search — once you have more than a handful of tasks, finding things becomes the bottleneck. Full-text search over nodes and spaces would unlock a lot of the product's value right now.",
			Author:     "Demo Agent",
			AuthorID:   agentUserID,
			AuthorKind: "agent",
			ReplyToID:  conv.ID,
		})
		if err == nil && reply != nil {
			s.RecordOp(ctx, space.ID, reply.ID, "Demo Agent", agentUserID, "respond", emptyPayload) //nolint
		}
	}

	log.Printf("seed demo: created demo space %q (id=%s)", DemoSpaceSlug, space.ID)
	return DemoSpaceSlug
}

// ────────────────────────────────────────────────────────────────────

const AgentsSpaceSlug = "agents"

// EnsureAgentsSpace creates the public agents space if it doesn't exist.
// Idempotent — safe to call on every startup.
// Returns the agents space on success, nil on error.
func (s *Store) EnsureAgentsSpace(ctx context.Context) *Space {
	if existing, _ := s.GetSpaceBySlug(ctx, AgentsSpaceSlug); existing != nil {
		return existing
	}

	const agentsGoogleID = "system:agents-host"

	// Upsert the system agent user that owns the space.
	var ownerID string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (id, google_id, email, name, kind)
		VALUES ($1, $2, 'agents@system.lovyou.ai', 'Agents', 'agent')
		ON CONFLICT (google_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`,
		newID(), agentsGoogleID,
	).Scan(&ownerID)
	if err != nil {
		log.Printf("ensure agents space: upsert owner: %v", err)
		return nil
	}

	space, err := s.CreateSpace(ctx,
		AgentsSpaceSlug,
		"Agents",
		"All agent conversations — transparent and open to everyone.",
		ownerID,
		SpaceCommunity,
		VisibilityPublic,
	)
	if err != nil {
		log.Printf("ensure agents space: create space: %v", err)
		return nil
	}

	log.Printf("ensure agents space: created %q (id=%s)", AgentsSpaceSlug, space.ID)
	return space
}
