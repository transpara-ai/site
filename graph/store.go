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
	Author       string     `json:"author"`
	AuthorKind   string     `json:"author_kind"` // "human" or "agent"
	Tags         []string   `json:"tags"`
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
	Actor     string          `json:"actor"`
	ActorKind string          `json:"actor_kind"` // "human" or "agent", resolved from users table
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
	Author     string
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
func (s *Store) ListPublicSpaces(ctx context.Context) ([]SpaceWithStats, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.id, s.slug, s.name, s.description, s.owner_id, s.kind, s.visibility, s.created_at,
		        COALESCE(n.cnt, 0), n.last_at,
		        COALESCE(m.member_count, 0), COALESCE(m.has_agent, false)
		 FROM spaces s
		 LEFT JOIN LATERAL (
		     SELECT COUNT(*) AS cnt, MAX(created_at) AS last_at
		     FROM nodes WHERE space_id = s.id
		 ) n ON true
		 LEFT JOIN LATERAL (
		     SELECT COUNT(DISTINCT o.actor) AS member_count,
		            BOOL_OR(u.kind = 'agent') AS has_agent
		     FROM ops o
		     LEFT JOIN users u ON u.name = o.actor
		     WHERE o.space_id = s.id
		 ) m ON true
		 WHERE s.visibility = 'public'
		 ORDER BY COALESCE(n.last_at, s.created_at) DESC`)
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
		Author:     p.Author,
		AuthorKind: authorKind,
		Tags:       p.Tags,
		DueDate:    p.DueDate,
	}

	var parentID *string
	if p.ParentID != "" {
		parentID = &p.ParentID
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO nodes (id, space_id, parent_id, kind, title, body, state, priority, assignee, author, author_kind, tags, due_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING created_at, updated_at`,
		n.ID, n.SpaceID, parentID, n.Kind, n.Title, n.Body, n.State, n.Priority,
		n.Assignee, n.Author, n.AuthorKind, pq.Array(n.Tags), n.DueDate,
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
		       n.state, n.priority, n.assignee, n.author, n.author_kind, n.tags, n.due_date,
		       n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       0
		FROM nodes n WHERE n.id = $1`, id,
	).Scan(
		&n.ID, &n.SpaceID, &parentID, &n.Kind, &n.Title, &n.Body,
		&n.State, &n.Priority, &n.Assignee, &n.Author, &n.AuthorKind, pq.Array(&n.Tags), &dueDate,
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
		       n.state, n.priority, n.assignee, n.author, n.author_kind, n.tags, n.due_date,
		       n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id), 0),
		       COALESCE((SELECT COUNT(*) FROM nodes c WHERE c.parent_id = n.id AND c.state = 'done'), 0),
		       0
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

	query += " ORDER BY n.created_at"

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
			&n.State, &n.Priority, &n.Assignee, &n.Author, &n.AuthorKind, pq.Array(&n.Tags), &dueDate,
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
func (s *Store) ListConversations(ctx context.Context, spaceID, userName string) ([]ConversationSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.space_id, n.parent_id, n.kind, n.title, n.body,
		       n.state, n.priority, n.assignee, n.author, n.author_kind, n.tags, n.due_date,
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
		  AND ($2 = ANY(n.tags) OR n.author = $2)
		ORDER BY n.updated_at DESC`, spaceID, userName)
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
			&cs.State, &cs.Priority, &cs.Assignee, &cs.Author, &cs.AuthorKind, pq.Array(&cs.Tags), &dueDate,
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

// HasAgentParticipant checks if any of the given names belong to an agent user.
func (s *Store) HasAgentParticipant(ctx context.Context, names []string) (bool, error) {
	if len(names) == 0 {
		return false, nil
	}
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE name = ANY($1) AND kind = 'agent')`,
		pq.Array(names)).Scan(&exists)
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
func (s *Store) UpdateNode(ctx context.Context, id string, title, body, priority, assignee *string) error {
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
func (s *Store) RecordOp(ctx context.Context, spaceID, nodeID, actor, op string, payload json.RawMessage) (*Op, error) {
	if payload == nil {
		payload = json.RawMessage(`{}`)
	}
	o := &Op{
		ID:      newID(),
		SpaceID: spaceID,
		NodeID:  nodeID,
		Actor:   actor,
		Op:      op,
		Payload: payload,
	}

	var nodeRef *string
	if nodeID != "" {
		nodeRef = &nodeID
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO ops (id, space_id, node_id, actor, op, payload) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
		o.ID, o.SpaceID, nodeRef, o.Actor, o.Op, o.Payload,
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
		`SELECT o.id, o.space_id, COALESCE(o.node_id, ''), o.actor,
		        COALESCE(u.kind, 'human'), o.op, o.payload, o.created_at
		 FROM ops o
		 LEFT JOIN users u ON u.name = o.actor
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
		if err := rows.Scan(&o.ID, &o.SpaceID, &o.NodeID, &o.Actor, &o.ActorKind, &o.Op, &o.Payload, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan op: %w", err)
		}
		ops = append(ops, o)
	}
	return ops, rows.Err()
}

// ListNodeOps returns operations for a specific node.
func (s *Store) ListNodeOps(ctx context.Context, nodeID string) ([]Op, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.id, o.space_id, COALESCE(o.node_id, ''), o.actor,
		        COALESCE(u.kind, 'human'), o.op, o.payload, o.created_at
		 FROM ops o
		 LEFT JOIN users u ON u.name = o.actor
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
		if err := rows.Scan(&o.ID, &o.SpaceID, &o.NodeID, &o.Actor, &o.ActorKind, &o.Op, &o.Payload, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan op: %w", err)
		}
		ops = append(ops, o)
	}
	return ops, rows.Err()
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
