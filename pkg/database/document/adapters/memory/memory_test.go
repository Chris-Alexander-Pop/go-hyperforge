package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/document"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/document/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryDocument_CRUD(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	defer store.Close()

	var _ document.Interface = store

	err := store.Insert(ctx, "users", document.Document{"id": "1", "name": "Ada"})
	require.NoError(t, err)

	found, err := store.Find(ctx, "users", map[string]interface{}{"name": "Ada"})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "1", found[0]["id"])

	err = store.Update(ctx, "users", map[string]interface{}{"id": "1"}, map[string]interface{}{"name": "Ada Lovelace"})
	require.NoError(t, err)

	found, err = store.Find(ctx, "users", map[string]interface{}{"id": "1"})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "Ada Lovelace", found[0]["name"])

	err = store.Delete(ctx, "users", map[string]interface{}{"id": "1"})
	require.NoError(t, err)

	found, err = store.Find(ctx, "users", map[string]interface{}{"id": "1"})
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestMemoryDocument_CollectionNotFound(t *testing.T) {
	store := memory.New()
	defer store.Close()

	_, err := store.Find(context.Background(), "missing", nil)
	require.Error(t, err)
}
