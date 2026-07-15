package testsuite

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/document"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// DocumentSuite is a reusable conformance suite for document.Interface implementations.
type DocumentSuite struct {
	*test.Suite
	Store document.Interface
	// Optional cleanup after each test.
	Cleanup func()
}

func (s *DocumentSuite) TearDownTest() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

func (s *DocumentSuite) TestCRUD() {
	ctx := context.Background()
	collection := "users"

	err := s.Store.Insert(ctx, collection, document.Document{"id": "1", "name": "Ada"})
	s.NoError(err)

	found, err := s.Store.Find(ctx, collection, map[string]interface{}{"name": "Ada"})
	s.NoError(err)
	s.Len(found, 1)
	s.Equal("1", found[0]["id"])

	err = s.Store.Update(ctx, collection, map[string]interface{}{"id": "1"}, map[string]interface{}{"name": "Ada Lovelace"})
	s.NoError(err)

	found, err = s.Store.Find(ctx, collection, map[string]interface{}{"id": "1"})
	s.NoError(err)
	s.Len(found, 1)
	s.Equal("Ada Lovelace", found[0]["name"])

	err = s.Store.Delete(ctx, collection, map[string]interface{}{"id": "1"})
	s.NoError(err)

	found, err = s.Store.Find(ctx, collection, map[string]interface{}{"id": "1"})
	s.NoError(err)
	s.Empty(found)
}

func (s *DocumentSuite) TestFindMultiple() {
	ctx := context.Background()
	collection := "items"

	s.NoError(s.Store.Insert(ctx, collection, document.Document{"id": "a", "type": "widget"}))
	s.NoError(s.Store.Insert(ctx, collection, document.Document{"id": "b", "type": "widget"}))
	s.NoError(s.Store.Insert(ctx, collection, document.Document{"id": "c", "type": "gadget"}))

	found, err := s.Store.Find(ctx, collection, map[string]interface{}{"type": "widget"})
	s.NoError(err)
	s.Len(found, 2)
}

func (s *DocumentSuite) TestDeleteMissingIsNoop() {
	ctx := context.Background()
	err := s.Store.Delete(ctx, "empty-collection", map[string]interface{}{"id": "x"})
	s.NoError(err)
}

func (s *DocumentSuite) TestClose() {
	s.NoError(s.Store.Close())
	s.Cleanup = nil
}
