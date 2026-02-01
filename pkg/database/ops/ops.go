package ops

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BulkUpsert ensures multiple records are created or updated.
// Uses GORM's Clauses for "ON CONFLICT UPDATE".
func BulkUpsert(ctx context.Context, db *gorm.DB, data interface{}, conflicts clause.OnConflict) error {
	return db.WithContext(ctx).Clauses(conflicts).Create(data).Error
}

// BulkDelete deletes records matching the given IDs.
// Model MUST be provided for GORM to know the table and logic (e.g. soft delete).
func BulkDelete(ctx context.Context, db *gorm.DB, model interface{}, ids []interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	// "id" hardcoded assumption? Or should we accept a conditional?
	// For "overengineered" generic, let's assume standard "id" column or GORM primary key convention.
	return db.WithContext(ctx).Delete(model, ids).Error
}

// ExecuteRaw runs a raw SQL command with arguments.
// Useful for DDL or complex migrations not supported by GORM.
func ExecuteRaw(ctx context.Context, db *gorm.DB, sql string, values ...interface{}) error {
	return db.WithContext(ctx).Exec(sql, values...).Error
}

// Transaction executes a function within a transaction.
func Transaction(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
