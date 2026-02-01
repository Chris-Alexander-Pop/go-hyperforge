package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/database/introspection"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/ops"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/transfer"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	"gorm.io/gorm/clause"
)

type OpsSuite struct {
	*test.Suite
}

func TestOpsSuite(t *testing.T) {
	test.Run(t, &OpsSuite{Suite: test.NewSuite()})
}

// User is a simple model for testing
type User struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Email string `gorm:"uniqueIndex"`
}

func (s *OpsSuite) TestBulkOps() {
	cfg := sql.Config{Name: "bulk_ops_test"}
	// Factory returns *gorm.DB directly now
	db, err := SqliteFactory(cfg)
	s.Require().NoError(err)

	s.Require().NoError(db.AutoMigrate(&User{}))

	ctx := context.Background()

	// 1. Bulk Upsert
	users := []User{
		{Name: "Alice", Email: "alice@example.com"}, // New
		{Name: "Bob", Email: "bob@example.com"},     // New
	}
	// "OnConflict" for email provided by uniqueIndex
	err = ops.BulkUpsert(ctx, db, &users, clause.OnConflict{
		DoNothing: true, // Simple case, or UpdateAll: true
	})
	s.NoError(err)

	var count int64
	db.Model(&User{}).Count(&count)
	s.Equal(int64(2), count)

	// 2. Introspection
	tables, err := introspection.ListTables(ctx, db)
	s.NoError(err)
	s.Contains(tables, "users")

	hasTable := introspection.HasTable(ctx, db, "users")
	s.True(hasTable)

	// 3. Transfer (Copy Table)
	// Create another DB
	destCfg := sql.Config{Name: "bulk_ops_dest"}
	destDB, err := SqliteFactory(destCfg)
	s.Require().NoError(err)

	// Copy users from db -> destDB
	err = transfer.CopyTable(ctx, db, destDB, &User{}, transfer.TransferOptions{
		BatchSize:  10,
		OnConflict: transfer.ConflictStrategyDoNothing,
	})
	s.NoError(err)

	// Verify dest has data
	var destCount int64
	destDB.Model(&User{}).Count(&destCount)
	s.Equal(int64(2), destCount)

	// 4. Bulk Delete
	// Retrieve IDs (sqlite auto-inc)
	var allUsers []User
	db.Find(&allUsers)

	ids := make([]interface{}, len(allUsers))
	for i, u := range allUsers {
		ids[i] = u.ID
	}

	err = ops.BulkDelete(ctx, db, &User{}, ids)
	s.NoError(err)

	var finalCount int64
	db.Model(&User{}).Count(&finalCount)
	s.Equal(int64(0), finalCount)
}
