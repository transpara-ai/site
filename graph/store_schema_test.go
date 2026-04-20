package graph

import (
	"context"
	"testing"
)

// TestOpsSchemaHiveChainRef ensures the hive_chain_ref column exists and is
// nullable. This guards the migration from being accidentally dropped.
func TestOpsSchemaHiveChainRef(t *testing.T) {
	db, _ := testDB(t)
	ctx := context.Background()

	var dataType string
	var isNullable string
	err := db.QueryRowContext(ctx, `
		SELECT data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'ops' AND column_name = 'hive_chain_ref'
	`).Scan(&dataType, &isNullable)
	if err != nil {
		t.Fatalf("query hive_chain_ref column: %v", err)
	}
	if dataType != "text" {
		t.Errorf("data_type = %q, want text", dataType)
	}
	if isNullable != "YES" {
		t.Errorf("is_nullable = %q, want YES", isNullable)
	}
}
