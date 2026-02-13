package firestore

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/document"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"google.golang.org/api/iterator"
)

// NOTE: This implementation assumes the standard GCP Firestore SDK is available.
// If not, please run: go get cloud.google.com/go/firestore

// Adapter implements the document.Interface for GCP Firestore.
type Adapter struct {
	client *firestore.Client
}

// New creates a new Firestore adapter.
func New(cfg document.Config) (*Adapter, error) {
	ctx := context.Background()
	// Firestore client automatically detects credentials (GOOGLE_APPLICATION_CREDENTIALS)
	client, err := firestore.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, errors.Internal("failed to create firestore client", err)
	}

	return &Adapter{
		client: client,
	}, nil
}

// Insert adds a new document.
func (a *Adapter) Insert(ctx context.Context, collection string, doc document.Document) error {
	// Firestore requires a document ID or it generates one.
	// If 'id' key exists in doc, use it.

	id, ok := doc["id"].(string)
	if ok {
		_, err := a.client.Collection(collection).Doc(id).Set(ctx, doc)
		if err != nil {
			return errors.Internal("failed to set firestore document", err)
		}
	} else {
		// Auto-generate ID
		_, _, err := a.client.Collection(collection).Add(ctx, doc)
		if err != nil {
			return errors.Internal("failed to add firestore document", err)
		}
	}
	return nil
}

// Find retrieves documents.
func (a *Adapter) Find(ctx context.Context, collection string, query map[string]interface{}) ([]document.Document, error) {
	q := a.client.Collection(collection).Query

	for k, v := range query {
		q = q.Where(k, "==", v)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var docs []document.Document
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Internal("failed to iterate firestore documents", err)
		}

		d := doc.Data()
		// Inject ID into document if not present, often useful
		if _, ok := d["id"]; !ok {
			d["id"] = doc.Ref.ID
		}

		docs = append(docs, document.Document(d))
	}

	return docs, nil
}

// Update modifies documents.
func (a *Adapter) Update(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) error {
	// Firestore update is by ID.
	id, ok := filter["id"].(string)
	if !ok {
		// If no ID is provided, we might need to find first... but standard Update API usually targets ID.
		// Implementing "Query and Update" is expensive.
		return errors.InvalidArgument("filter must contain 'id' for update", nil)
	}

	updates := make([]firestore.Update, 0, len(update))
	for k, v := range update {
		// Split nested paths if dot notation is used
		path := k
		// Assuming simple update for now, real impl might handle dot notation for nested fields

		updates = append(updates, firestore.Update{
			Path:  path,
			Value: v,
		})
	}

	_, err := a.client.Collection(collection).Doc(id).Update(ctx, updates)
	if err != nil {
		if isNotFound(err) {
			return errors.NotFound("document not found", err)
		}
		return errors.Internal("failed to update firestore document", err)
	}

	return nil
}

// Delete removes documents.
func (a *Adapter) Delete(ctx context.Context, collection string, filter map[string]interface{}) error {
	id, ok := filter["id"].(string)
	if !ok {
		return errors.InvalidArgument("filter must contain 'id' for delete", nil)
	}

	_, err := a.client.Collection(collection).Doc(id).Delete(ctx)
	if err != nil {
		return errors.Internal("failed to delete firestore document", err)
	}
	return nil
}

// Close closes the Firestore client.
func (a *Adapter) Close() error {
	return a.client.Close()
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "not found")
}
