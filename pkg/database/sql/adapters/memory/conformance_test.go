package memory_test

import (
	"fmt"
	"sync/atomic"
	"testing"

	dbsql "github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/sql/testsuite"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

var sqlConformanceSeq atomic.Uint64

type MemorySuite struct {
	testsuite.SQLSuite
}

func (s *MemorySuite) SetupTest() {
	s.Suite.SetupTest()
	name := fmt.Sprintf("sql_conformance_%d", sqlConformanceSeq.Add(1))
	db, err := memory.NewWithConfig(dbsql.Config{Name: name})
	s.Require().NoError(err)
	s.Store = db
	s.Cleanup = func() {
		if s.Store != nil {
			_ = s.Store.Close()
		}
	}
}

func TestMemorySQLConformance(t *testing.T) {
	test.Run(t, &MemorySuite{SQLSuite: testsuite.SQLSuite{Suite: test.NewSuite()}})
}
