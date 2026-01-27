package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/data/search"
	"github.com/chris-alexander-pop/system-design-library/pkg/data/search/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// SearchEngineSuite provides a generic test suite for SearchEngine implementations.
type SearchEngineSuite struct {
	suite.Suite
	engine search.SearchEngine
	ctx    context.Context
}

// SetupTest runs before each test.
func (s *SearchEngineSuite) SetupTest() {
	s.engine = memory.New()
	s.ctx = context.Background()
}

func (s *SearchEngineSuite) TearDownTest() {
	if s.engine != nil {
		s.engine.Close()
	}
}

func (s *SearchEngineSuite) TestCreateAndDeleteIndex() {
	err := s.engine.CreateIndex(s.ctx, "test-index", &search.IndexMapping{
		Fields: map[string]search.FieldMapping{
			"title": {Type: search.FieldTypeText, Searchable: true},
			"price": {Type: search.FieldTypeFloat, Filterable: true},
		},
	})
	s.Require().NoError(err)

	info, err := s.engine.GetIndex(s.ctx, "test-index")
	s.Require().NoError(err)
	s.Equal("test-index", info.Name)

	err = s.engine.DeleteIndex(s.ctx, "test-index")
	s.Require().NoError(err)

	_, err = s.engine.GetIndex(s.ctx, "test-index")
	s.Error(err)
}

func (s *SearchEngineSuite) TestCreateIndexAlreadyExists() {
	err := s.engine.CreateIndex(s.ctx, "duplicate", nil)
	s.Require().NoError(err)

	err = s.engine.CreateIndex(s.ctx, "duplicate", nil)
	s.Error(err)
}

func (s *SearchEngineSuite) TestIndexAndGet() {
	doc := map[string]interface{}{
		"title": "Test Document",
		"body":  "This is a test",
	}

	err := s.engine.Index(s.ctx, "products", "doc1", doc)
	s.Require().NoError(err)

	hit, err := s.engine.Get(s.ctx, "products", "doc1")
	s.Require().NoError(err)
	s.Equal("doc1", hit.ID)
	s.Equal("Test Document", hit.Source["title"])
}

func (s *SearchEngineSuite) TestGetNotFound() {
	err := s.engine.CreateIndex(s.ctx, "empty", nil)
	s.Require().NoError(err)

	_, err = s.engine.Get(s.ctx, "empty", "nonexistent")
	s.Error(err)
}

func (s *SearchEngineSuite) TestDelete() {
	doc := map[string]interface{}{"title": "Delete me"}

	err := s.engine.Index(s.ctx, "products", "to-delete", doc)
	s.Require().NoError(err)

	err = s.engine.Delete(s.ctx, "products", "to-delete")
	s.Require().NoError(err)

	_, err = s.engine.Get(s.ctx, "products", "to-delete")
	s.Error(err)
}

func (s *SearchEngineSuite) TestSearch() {
	// Index some documents
	docs := []struct {
		id  string
		doc map[string]interface{}
	}{
		{"1", map[string]interface{}{"title": "Apple iPhone", "category": "phones", "price": 999}},
		{"2", map[string]interface{}{"title": "Samsung Galaxy", "category": "phones", "price": 899}},
		{"3", map[string]interface{}{"title": "Apple MacBook", "category": "laptops", "price": 1999}},
		{"4", map[string]interface{}{"title": "Dell XPS", "category": "laptops", "price": 1499}},
	}

	for _, d := range docs {
		err := s.engine.Index(s.ctx, "products", d.id, d.doc)
		s.Require().NoError(err)
	}

	// Search for Apple products
	result, err := s.engine.Search(s.ctx, "products", search.Query{
		Text: "Apple",
	})
	s.Require().NoError(err)
	s.Equal(int64(2), result.Total)
}

func (s *SearchEngineSuite) TestSearchWithFilters() {
	// Index documents
	docs := []struct {
		id  string
		doc map[string]interface{}
	}{
		{"1", map[string]interface{}{"name": "Product A", "price": 100.0}},
		{"2", map[string]interface{}{"name": "Product B", "price": 200.0}},
		{"3", map[string]interface{}{"name": "Product C", "price": 300.0}},
	}

	for _, d := range docs {
		err := s.engine.Index(s.ctx, "products", d.id, d.doc)
		s.Require().NoError(err)
	}

	// Filter by price > 150
	result, err := s.engine.Search(s.ctx, "products", search.Query{
		Filters: []search.Filter{
			{Field: "price", Operator: search.FilterOperatorGreaterThan, Value: 150.0},
		},
	})
	s.Require().NoError(err)
	s.Equal(int64(2), result.Total)
}

func (s *SearchEngineSuite) TestSearchWithPagination() {
	// Index many documents
	for i := 0; i < 20; i++ {
		doc := map[string]interface{}{"name": "Product", "index": i}
		err := s.engine.Index(s.ctx, "products", string(rune('a'+i)), doc)
		s.Require().NoError(err)
	}

	// First page
	result, err := s.engine.Search(s.ctx, "products", search.Query{
		From: 0,
		Size: 5,
	})
	s.Require().NoError(err)
	s.Len(result.Hits, 5)

	// Second page
	result, err = s.engine.Search(s.ctx, "products", search.Query{
		From: 5,
		Size: 5,
	})
	s.Require().NoError(err)
	s.Len(result.Hits, 5)
}

func (s *SearchEngineSuite) TestSearchWithFacets() {
	// Index documents with categories
	docs := []struct {
		id  string
		doc map[string]interface{}
	}{
		{"1", map[string]interface{}{"name": "iPhone", "category": "phones"}},
		{"2", map[string]interface{}{"name": "Galaxy", "category": "phones"}},
		{"3", map[string]interface{}{"name": "MacBook", "category": "laptops"}},
		{"4", map[string]interface{}{"name": "XPS", "category": "laptops"}},
		{"5", map[string]interface{}{"name": "iPad", "category": "tablets"}},
	}

	for _, d := range docs {
		err := s.engine.Index(s.ctx, "products", d.id, d.doc)
		s.Require().NoError(err)
	}

	// Search with facets
	result, err := s.engine.Search(s.ctx, "products", search.Query{
		Facets: []string{"category"},
	})
	s.Require().NoError(err)
	s.NotNil(result.Facets)
	s.Contains(result.Facets, "category")

	categories := result.Facets["category"]
	s.NotEmpty(categories)
}

func (s *SearchEngineSuite) TestBulk() {
	ops := []search.BulkOperation{
		{Action: search.BulkActionIndex, ID: "bulk1", Document: map[string]interface{}{"name": "Bulk 1"}},
		{Action: search.BulkActionIndex, ID: "bulk2", Document: map[string]interface{}{"name": "Bulk 2"}},
		{Action: search.BulkActionIndex, ID: "bulk3", Document: map[string]interface{}{"name": "Bulk 3"}},
	}

	result, err := s.engine.Bulk(s.ctx, "products", ops)
	s.Require().NoError(err)
	s.Equal(3, result.Successful)
	s.Equal(0, result.Failed)

	// Verify documents exist
	for _, op := range ops {
		hit, err := s.engine.Get(s.ctx, "products", op.ID)
		s.Require().NoError(err)
		s.NotNil(hit)
	}

	// Bulk delete
	deleteOps := []search.BulkOperation{
		{Action: search.BulkActionDelete, ID: "bulk1"},
		{Action: search.BulkActionDelete, ID: "bulk2"},
	}

	result, err = s.engine.Bulk(s.ctx, "products", deleteOps)
	s.Require().NoError(err)
	s.Equal(2, result.Successful)

	// Verify deleted
	_, err = s.engine.Get(s.ctx, "products", "bulk1")
	s.Error(err)

	// Verify bulk3 still exists
	_, err = s.engine.Get(s.ctx, "products", "bulk3")
	s.NoError(err)
}

func (s *SearchEngineSuite) TestRefresh() {
	err := s.engine.Index(s.ctx, "products", "doc1", map[string]interface{}{"name": "Test"})
	s.Require().NoError(err)

	err = s.engine.Refresh(s.ctx, "products")
	s.NoError(err)
}

func (s *SearchEngineSuite) TestSearchTiming() {
	// Index a document
	err := s.engine.Index(s.ctx, "products", "1", map[string]interface{}{"name": "Test"})
	s.Require().NoError(err)

	result, err := s.engine.Search(s.ctx, "products", search.Query{Text: "Test"})
	s.Require().NoError(err)
	s.True(result.Took > 0 || result.Took == 0) // Memory is instant
	s.True(result.Took < time.Second)           // Should be fast
}

// TestSearchEngineSuite runs the test suite.
func TestSearchEngineSuite(t *testing.T) {
	suite.Run(t, new(SearchEngineSuite))
}
