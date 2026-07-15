package testsuite

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// VectorSuite is a reusable conformance suite for vector.Store implementations.
type VectorSuite struct {
	*test.Suite
	Store vector.Store
	// Optional cleanup after each test.
	Cleanup func()
}

func (s *VectorSuite) TearDownTest() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

func (s *VectorSuite) TestUpsertSearchDelete() {
	ctx := context.Background()

	s.NoError(s.Store.Upsert(ctx, "a", []float32{1, 0}, map[string]interface{}{"tag": "x"}))
	s.NoError(s.Store.Upsert(ctx, "b", []float32{0, 1}, map[string]interface{}{"tag": "y"}))

	results, err := s.Store.Search(ctx, []float32{1, 0}, 2)
	s.NoError(err)
	s.NotEmpty(results)
	s.Equal("a", results[0].ID)

	s.NoError(s.Store.Delete(ctx, "a"))

	err = s.Store.Delete(ctx, "a")
	s.Error(err)
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		s.Equal(errors.CodeNotFound, appErr.Code)
	}
}

func (s *VectorSuite) TestSearchWithOptsFilter() {
	ctx := context.Background()

	s.NoError(s.Store.Upsert(ctx, "a", []float32{1, 0}, map[string]interface{}{"lang": "en", "tier": 1}))
	s.NoError(s.Store.Upsert(ctx, "b", []float32{0.9, 0.1}, map[string]interface{}{"lang": "fr", "tier": 1}))
	s.NoError(s.Store.Upsert(ctx, "c", []float32{0.8, 0.2}, map[string]interface{}{"lang": "en", "tier": 2}))

	results, err := s.Store.SearchWithOpts(ctx, []float32{1, 0}, vector.SearchOpts{
		Limit:  10,
		Filter: map[string]interface{}{"lang": "en"},
	})
	s.NoError(err)
	s.Len(results, 2)
	for _, r := range results {
		s.Equal("en", r.Metadata["lang"])
	}

	results, err = s.Store.SearchWithOpts(ctx, []float32{1, 0}, vector.SearchOpts{
		Limit:  10,
		Filter: map[string]interface{}{"lang": "en", "tier": 2},
	})
	s.NoError(err)
	s.Len(results, 1)
	s.Equal("c", results[0].ID)
}

func (s *VectorSuite) TestUpsertOverwrite() {
	ctx := context.Background()

	s.NoError(s.Store.Upsert(ctx, "v1", []float32{1, 0}, map[string]interface{}{"v": 1}))
	s.NoError(s.Store.Upsert(ctx, "v1", []float32{0, 1}, map[string]interface{}{"v": 2}))

	results, err := s.Store.Search(ctx, []float32{0, 1}, 1)
	s.NoError(err)
	s.Require().NotEmpty(results)
	s.Equal("v1", results[0].ID)
	s.Equal(2, results[0].Metadata["v"])
}

func (s *VectorSuite) TestEmptySearch() {
	ctx := context.Background()
	results, err := s.Store.Search(ctx, []float32{1, 0}, 5)
	s.NoError(err)
	s.Empty(results)
}

func (s *VectorSuite) TestClose() {
	s.NoError(s.Store.Close())
	s.Cleanup = nil
}
