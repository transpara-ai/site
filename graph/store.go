// Package graph implements the unified product backend — spaces, nodes, and
// grammar operations backed by Postgres. Replaces the separate work/social/market
// packages with one data model where every action is a grammar operation.
package graph

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// ────────────────────────────────────────────────────────────────────
// Domain types
// ────────────────────────────────────────────────────────────────────

// Node states.
const (
	StateOpen   = "open"
	StateActive = "active"
	StateReview = "review"
	StateDone   = "done"
	StateClosed = "closed"
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

// Space is a container — project, community, or team.
type Space struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     string    `json:"owner_id"`
	Kind        string    `json:"kind"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
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
	Author       string     `json:"author"`
	AuthorID     string     `json:"author_id"`               // user ID — source of truth for identity
	AuthorKind   string     `json:"author_kind"`              // "human" or "agent"
	Tags         []string   `json:"tags"`
	Pinned       bool       `json:"pinned"`
	DueDate      *time.Time `json:"due_date,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ChildCount   int        `json:"child_count"`
	ChildDone    int        `json:"child_done"`
	BlockerCount int        `json:"blocker_count"`
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
}

// ListNodesParams controls filtering for node listing.
type ListNodesParams struct {
	SpaceID  string
	Kind     string     // filter by kind, empty = all
	State    string     // filter by state, empty = all
	ParentID string     // "root" = top-level only, ID = children, empty = all
	After    *time.Time // only nodes created after this time, nil = all
	Pinned   bool       // if true, only return pinned nodes
	Limit    int        // max results, 0 = default (500)
}

// ────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────

var (
	ErrNotFound = errors.New("not found")
)

// ────────────────────────────────────────────────────────────────────
// Store
// ────────────────────────────────────────────────────────────────────

// Store is a Postgres-backed store for the unified product.
type Store struct {
	db *sql.DB
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

CREATE TABLE IF NOT EXISTS endorsements (
    from_id    TEXT NOT NULL,
    to_id      TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (from_id, to_id)
);
CREATE INDEX IF NOT EXISTS idx_endorsements_to ON endorsements(to_id);

-- Backfill assignee_id from users table where assignee name matches.
UPDATE nodes SET assignee_id = u.id
FROM users u WHERE nodes.assignee = u.name AND nodes.assignee_id = '' AND nodes.assignee != '';

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
		`SELECT id, slug, name, description, owner_id, kind, visibility, created_at
		 FROM spaces WHERE owner_id = $1 ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}
	defer rows.Close()

	var spaces []Space
	for rows.Next() {
		var sp Space
		if err := rows.Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.CreatedAt); err != nil {
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
	q := `SELECT s.id, s.slug, s.name, s.description, s.owner_id, s.kind, s.visibility, s.created_at,
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
		 WHERE s.visibility = 'public'`
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
		if err := rows.Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.CreatedAt,
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
		`SELECT id, slug, name, description, owner_id, kind, visibility, created_at FROM spaces WHERE id = $1`, id,
	).Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.CreatedAt)
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
		`SELECT id, slug, name, description, owner_id, kind, visibility, created_at FROM spaces WHERE slug = $1`, slug,
	).Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get space: %w", err)
	}
	return &sp, nil
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
	}

	var parentID *string
	if p.ParentID != "" {
		parentID = &p.ParentID
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO nodes (id, space_id, parent_id, kind, title, body, state, priority, assignee, assignee_id, author, author_id, author_kind, tags, due_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING created_at, updated_at`,
		n.ID, n.SpaceID, parentID, n.Kind, n.Title, n.Body, n.State, n.Priority,
		n.Assignee, n.AssigneeID, n.Author, n.AuthorID, n.AuthorKind, pq.Array(n.Tags), n.DueDate,
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
		       n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM node_deps d JOIN nodes b ON b.id = d.depends_on WHERE d.node_id = n.id AND b.state != 'done'), 0)
		FROM nodes n WHERE n.id = $1`, id,
	).Scan(
		&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
		&n.State, &n.Priority, &n.Assignee, &n.AssigneeID, &n.Author, &n.AuthorID, &n.AuthorKind, pq.Array(&n.Tags), &n.Pinned, &dueDate,
		&n.CreatedAt, &n.UpdatedAt,
		&n.ChildCount, &n.ChildDone, &n.BlockerCount,
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
		       n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM node_deps d JOIN nodes b ON b.id = d.depends_on WHERE d.node_id = n.id AND b.state != 'done'), 0)
		FROM nodes n
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

	query += " ORDER BY n.pinned DESC, n.created_at"

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
			&n.CreatedAt, &n.UpdatedAt,
			&n.ChildCount, &n.ChildDone, &n.BlockerCount,
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

// ConversationSummary is a conversation node with last message preview.
type ConversationSummary struct {
	Node
	LastAuthor     string `json:"last_author,omitempty"`
	LastAuthorKind string `json:"last_author_kind,omitempty"`
	LastBody       string `json:"last_body,omitempty"`
}

// ListConversations returns conversations in a space that involve the given user.
// Matches on userID in tags or as author_id.
func (s *Store) ListConversations(ctx context.Context, spaceID, userID string) ([]ConversationSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, n.parent_id, n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.assignee_id, n.author, n.author_id, n.author_kind, n.tags, n.pinned, n.due_date,
		       n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       0, 0,
		       lm.author, lm.author_kind, lm.body
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
			&cs.CreatedAt, &cs.UpdatedAt,
			&cs.ChildCount, &cs.ChildDone, &cs.BlockerCount,
			&lastAuthor, &lastAuthorKind, &lastBody,
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

// UpdateNodeState sets a node's state.
func (s *Store) UpdateNodeState(ctx context.Context, id, state string) error {
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

// MemberCount returns the number of members in a space.
func (s *Store) MemberCount(ctx context.Context, spaceID string) int {
	var count int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM space_members WHERE space_id = $1`, spaceID).Scan(&count)
	return count
}

// GetUserProfile returns a user by name (for public profiles).
func (s *Store) GetUserProfile(ctx context.Context, name string) (*struct {
	ID        string
	Name      string
	Kind      string
	TasksDone int
	OpCount   int
}, error) {
	var u struct {
		ID        string
		Name      string
		Kind      string
		TasksDone int
		OpCount   int
	}
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.kind,
		       COALESCE((SELECT COUNT(*) FROM nodes n WHERE n.author_id = u.id AND n.kind = 'task' AND n.state = 'done'), 0),
		       COALESCE((SELECT COUNT(*) FROM ops o WHERE o.actor_id = u.id), 0)
		FROM users u WHERE u.name = $1`, name,
	).Scan(&u.ID, &u.Name, &u.Kind, &u.TasksDone, &u.OpCount)
	if err != nil {
		return nil, err
	}
	return &u, nil
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
		       n.tags, n.pinned, n.due_date, n.created_at, n.updated_at, 0, 0, 0
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
		SELECT n.id, n.user_id, n.op_id, n.space_id, n.message, n.read, n.created_at,
		       COALESCE(s.slug, ''), COALESCE(s.name, '')
		FROM notifications n
		LEFT JOIN spaces s ON s.id = n.space_id
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
		if err := rows.Scan(&n.ID, &n.UserID, &n.OpID, &n.SpaceID, &n.Message, &n.Read, &n.CreatedAt,
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

// ProposalWithVotes is a proposal node with vote tallies.
type ProposalWithVotes struct {
	Node
	VotesYes int `json:"votes_yes"`
	VotesNo  int `json:"votes_no"`
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
		       COALESCE((SELECT COUNT(*) FROM ops o WHERE o.node_id = n.id AND o.op = 'vote' AND o.payload->>'vote' = 'no'), 0)
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

// HasVoted checks if a user has already voted on a proposal.
func (s *Store) HasVoted(ctx context.Context, nodeID, actorID string) bool {
	var exists bool
	s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM ops WHERE node_id = $1 AND actor_id = $2 AND op = 'vote')`,
		nodeID, actorID).Scan(&exists)
	return exists
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
		`SELECT id, slug, name, description, owner_id, kind, visibility, created_at
		 FROM spaces WHERE visibility = 'public' AND (name ILIKE $1 OR description ILIKE $1)
		 ORDER BY created_at DESC LIMIT $2`, pattern, limit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sp Space
			if rows.Scan(&sp.ID, &sp.Slug, &sp.Name, &sp.Description, &sp.OwnerID, &sp.Kind, &sp.Visibility, &sp.CreatedAt) == nil {
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

// CountChallenges returns the number of challenge ops for a given node.
func (s *Store) CountChallenges(ctx context.Context, nodeID string) int {
	var count int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ops WHERE node_id = $1 AND op = 'challenge'`, nodeID).Scan(&count)
	return count
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
