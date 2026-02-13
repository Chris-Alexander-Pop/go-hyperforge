package partitioning

import (
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"gorm.io/gorm"
)

// quoteIdentifier sanitizes a SQL identifier
func quoteIdentifier(s string) string {
	// Simple sanitization: double quote and escape internal double quotes
	// This is valid for Postgres/SQLite. MySQL uses backticks usually, but ANSI mode supports "".
	// For simplicity/standardization we assume Postgres-like.
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(s, "\"", "\"\""))
}

// quoteLiteral sanitizes a SQL string literal
func quoteLiteral(s string) string {
	// Simple sanitization: escape single quotes by doubling them
	return strings.ReplaceAll(s, "'", "''")
}

// CreateRangePartition creates a range partition on a table
// Example: CreateRangePartition(db, "orders", "created_at", "2023-01-01", "2023-02-01")
func CreateRangePartition(db *gorm.DB, table, column, start, end string) error {
	partitionName := fmt.Sprintf("%s_%s_%s", table, start, end)

	// Sanitize inputs
	qTable := quoteIdentifier(table)
	qPart := quoteIdentifier(partitionName)
	// qColumn := quoteIdentifier(column) // Not strictly needed in the CREATE TABLE syntax used below

	// Escape string literals to prevent SQL injection
	qStart := quoteLiteral(start)
	qEnd := quoteLiteral(end)

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s');",
		qPart, qTable, qStart, qEnd)

	if err := db.Exec(sql).Error; err != nil {
		return errors.Wrap(err, "failed to create range partition")
	}
	return nil
}

// CreateHashPartition creates a hash partition
// Example: CreateHashPartition(db, "users", "id", 3, 0) // Modulus 3, Remainder 0
func CreateHashPartition(db *gorm.DB, table, column string, modulus, remainder int) error {
	partitionName := fmt.Sprintf("%s_mod%d_rem%d", table, modulus, remainder)

	qTable := quoteIdentifier(table)
	qPart := quoteIdentifier(partitionName)

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES WITH (MODULUS %d, REMAINDER %d);",
		qPart, qTable, modulus, remainder)

	if err := db.Exec(sql).Error; err != nil {
		return errors.Wrap(err, "failed to create hash partition")
	}
	return nil
}
