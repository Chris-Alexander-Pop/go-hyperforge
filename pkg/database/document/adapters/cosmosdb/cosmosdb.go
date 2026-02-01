package cosmosdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/document"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// NOTE: This implementation assumes the Azure SDK for Go is available.
// If not, please run: go get github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos

// Adapter implements the document.Interface for Azure CosmosDB.
type Adapter struct {
	client   *azcosmos.Client
	database *azcosmos.DatabaseClient
}

// New creates a new CosmosDB adapter.
func New(cfg document.Config) (*Adapter, error) {
	// Construct connection string or use credential with Endpoint
	// Assuming Key based auth for simplicity here. Check documentation for DefaultAzureCredential

	keyCred, err := azcosmos.NewKeyCredential(cfg.Password) // Using Config Password as Key
	if err != nil {
		return nil, errors.Internal("failed to create cosmos key credential", err)
	}

	client, err := azcosmos.NewClientWithKey(cfg.Host, keyCred, nil)
	if err != nil {
		return nil, errors.Internal("failed to create cosmos client", err)
	}

	dbClient, err := client.NewDatabase(cfg.Database)
	if err != nil {
		return nil, errors.Internal("failed to create cosmos database client", err)
	}

	return &Adapter{
		client:   client,
		database: dbClient,
	}, nil
}

// Insert adds a new document.
func (a *Adapter) Insert(ctx context.Context, collection string, doc document.Document) error {
	container, err := a.database.NewContainer(collection)
	if err != nil {
		return errors.Internal("failed to get container client", err)
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return errors.Internal("failed to marshal document", err)
	}

	// Partition Key is mandatory. We assume the document has an "id" or specific partition key.
	// For generic impl, we need the user to tell us, or we extract it.
	// Assuming "id" is the partition key for simplicity in this generic adapter.
	pkVal, ok := doc["id"].(string)
	if !ok {
		return errors.InvalidArgument("document must have a string 'id' field acting as partition key", nil)
	}
	pk := azcosmos.NewPartitionKeyString(pkVal)

	_, err = container.CreateItem(ctx, pk, data, nil)
	if err != nil {
		return errors.Internal("failed to create item in cosmosdb", err)
	}
	return nil
}

// Find retrieves documents.
func (a *Adapter) Find(ctx context.Context, collection string, query map[string]interface{}) ([]document.Document, error) {
	container, err := a.database.NewContainer(collection)
	if err != nil {
		return nil, errors.Internal("failed to get container client", err)
	}

	queryText, queryParams := buildQuery(query)

	opt := &azcosmos.QueryOptions{
		QueryParameters: queryParams,
	}

	pager := container.NewQueryItemsPager(queryText, azcosmos.NewPartitionKeyString("TODO_partition_key"), opt)
	// Querying across partitions (cross-partition query) usually requires omit PK or specific options.
	// Current Generic Interface makes it hard to specify Partition Key separate from Query.

	// This code likely needs `azcosmos.QueryOptions{PopulateIndexMetrics: ...}` or similar adjustment for cross-partition.

	var docs []document.Document
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, errors.Internal("failed to query cosmos page", err)
		}

		for _, bytes := range resp.Items {
			var doc document.Document
			if err := json.Unmarshal(bytes, &doc); err != nil {
				return nil, errors.Internal("failed to unmarshal item", err)
			}
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// Update modifies documents.
func (a *Adapter) Update(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) error {
	// CosmosDB supports Patch operations.
	container, err := a.database.NewContainer(collection)
	if err != nil {
		return errors.Internal("failed to get container client", err)
	}

	id, ok := filter["id"].(string)
	if !ok {
		return errors.InvalidArgument("filter must contain 'id'", nil)
	}
	pk := azcosmos.NewPartitionKeyString(id)

	var operations azcosmos.PatchOperations
	for k, v := range update {
		operations.AppendSet(fmt.Sprintf("/%s", k), v)
	}

	_, err = container.PatchItem(ctx, pk, id, operations, nil)
	if err != nil {
		return errors.Internal("failed to patch item in cosmosdb", err)
	}

	return nil
}

// Delete removes documents.
func (a *Adapter) Delete(ctx context.Context, collection string, filter map[string]interface{}) error {
	container, err := a.database.NewContainer(collection)
	if err != nil {
		return errors.Internal("failed to get container client", err)
	}

	id, ok := filter["id"].(string)
	if !ok {
		return errors.InvalidArgument("filter must contain 'id'", nil)
	}
	pk := azcosmos.NewPartitionKeyString(id)

	_, err = container.DeleteItem(ctx, pk, id, nil)
	if err != nil {
		if responseErr, ok := err.(*azcore.ResponseError); ok && responseErr.StatusCode == 404 {
			return errors.NotFound("document not found", err)
		}
		return errors.Internal("failed to delete item from cosmosdb", err)
	}
	return nil
}

// Close is a no-op generic interface, client doesn't need explicit close usually.
func (a *Adapter) Close() error {
	return nil
}

func buildQuery(query map[string]interface{}) (string, []azcosmos.QueryParameter) {
	var sb strings.Builder
	sb.WriteString("SELECT * FROM c")

	if len(query) == 0 {
		return sb.String(), nil
	}

	sb.WriteString(" WHERE ")
	params := make([]azcosmos.QueryParameter, 0, len(query))
	i := 0
	for k, v := range query {
		if i > 0 {
			sb.WriteString(" AND ")
		}

		paramName := fmt.Sprintf("@p%d", i)
		sb.WriteString("c.")
		sb.WriteString(k)
		sb.WriteString(" = ")
		sb.WriteString(paramName)

		params = append(params, azcosmos.QueryParameter{
			Name:  paramName,
			Value: v,
		})
		i++
	}

	return sb.String(), params
}
