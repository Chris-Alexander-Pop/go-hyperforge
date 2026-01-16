package partitioning

import (
	"fmt"

	"gorm.io/gorm"
)

// CreateRangePartition creates a range partition on a table
// Example: CreateRangePartition(db, "orders", "created_at", "2023-01-01", "2023-02-01")
func CreateRangePartition(db *gorm.DB, table, column, start, end string) error {
	partitionName := fmt.Sprintf("%s_%s_%s", table, start, end)
	// Sanitize inputs in real app!
	// TBI: Robust SQL injection prevention for table/column names (using "" quoting)
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s');",
		partitionName, table, start, end)

	return db.Exec(sql).Error
}

// CreateHashPartition creates a hash partition
// Example: CreateHashPartition(db, "users", "id", 3, 0) // Modulus 3, Remainder 0
func CreateHashPartition(db *gorm.DB, table, column string, modulus, remainder int) error {
	partitionName := fmt.Sprintf("%s_mod%d_rem%d", table, modulus, remainder)
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES WITH (MODULUS %d, REMAINDER %d);",
		partitionName, table, modulus, remainder)

	return db.Exec(sql).Error
}
