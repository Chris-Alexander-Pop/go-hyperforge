package testsuite

import (
	"context"

	dbsql "github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// SQLSuite is a thin conformance suite for sql.SQL implementations.
// It verifies Get returns a usable DB and Close succeeds.
type SQLSuite struct {
	*test.Suite
	Store dbsql.SQL
	// Optional cleanup after each test.
	Cleanup func()
}

func (s *SQLSuite) TearDownTest() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

func (s *SQLSuite) TestGetReturnsDB() {
	ctx := context.Background()
	db := s.Store.Get(ctx)
	s.NotNil(db)

	sqlDB, err := db.DB()
	s.NoError(err)
	s.NoError(sqlDB.Ping())
}

func (s *SQLSuite) TestClose() {
	err := s.Store.Close()
	s.NoError(err)
	// Prevent TearDown Cleanup from closing twice (sql.DB.Close is not idempotent).
	s.Store = nil
	s.Cleanup = nil
}
