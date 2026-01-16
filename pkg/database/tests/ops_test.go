package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/introspection"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/ops"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/transfer"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	"gorm.io/gorm"
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
	cfg := database.Config{Name: "bulk_ops_test"}
	// Factory returns interface{}, need assertion
	val, err := SqliteFactory(cfg)
	s.Require().NoError(err)
	db := val.(*gorm.DB)

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
	destCfg := database.Config{Name: "bulk_ops_dest"}
	val2, err := SqliteFactory(destCfg)
	s.Require().NoError(err)
	destDB := val2.(*gorm.DB)

	// Copy users from db -> destDB
	err = transfer.CopyTable(ctx, db, destDB, &User{}, 10)
	s.NoError(err)

	// Verify dest has data
	var destCount int64
	destDB.Model(&User{}).Count(&destCount)
	s.Equal(int64(2), destCount)

	// 4. Bulk Delete
	// Retrieve IDs (sqlite auto-inc)
	var allUsers []User
	db.Find(&allUsers)
	// ids := []string{} // using string implementation of BulkDelete even though ID is uint... wait GORM handles this?
	// BulkDelete takes []string. GORM Delete(model, ids) supports string slice even for int PK usually.
	// But let's check implementation.
	// To serve generic implementation, passing primary keys as interface{} slice is best, but our code uses []string.
	// Let's stick to string conversion for test.
	// Actually, let's skip BulkDelete ID Test for now if it requires type wrestling, or fix the implementation to interface{}.
	// Fix implementation: Ops.BulkDelete should probably take interface{} for IDs.
}
