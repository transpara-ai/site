// cleanup-orphans closes all child nodes whose parent is done but the child is not.
// This is a one-time migration for the orphaned subtask chains created before cascade
// close was introduced.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/cleanup-orphans/
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Count distinct parent tasks that are done but have non-done children.
	var orphanParents int
	if err := db.QueryRow(`
		SELECT COUNT(DISTINCT parent_id)
		FROM nodes
		WHERE state != 'done'
		  AND parent_id IN (SELECT id FROM nodes WHERE state = 'done')
	`).Scan(&orphanParents); err != nil {
		log.Fatalf("count orphan parents: %v", err)
	}

	if orphanParents == 0 {
		fmt.Println("No orphaned subtask chains found — nothing to do.")
		return
	}
	fmt.Printf("Found %d parent tasks with orphaned children. Closing all descendants...\n", orphanParents)

	// Recursively close all non-done descendants of done parents.
	// The CTE walks the full subtree so nested orphans are caught in one pass.
	res, err := db.Exec(`
		WITH RECURSIVE orphans AS (
			-- Direct non-done children of done parents.
			SELECT c.id
			FROM nodes c
			JOIN nodes p ON c.parent_id = p.id
			WHERE p.state = 'done' AND c.state != 'done'
			UNION ALL
			-- Non-done descendants of those children.
			SELECT c.id
			FROM nodes c
			JOIN orphans o ON c.parent_id = o.id
			WHERE c.state != 'done'
		)
		UPDATE nodes
		SET state = 'done', updated_at = NOW()
		WHERE id IN (SELECT id FROM orphans)
	`)
	if err != nil {
		log.Fatalf("cleanup: %v", err)
	}

	n, _ := res.RowsAffected()
	fmt.Printf("Done. Closed %d orphaned subtasks across %d parent chains.\n", n, orphanParents)
}
