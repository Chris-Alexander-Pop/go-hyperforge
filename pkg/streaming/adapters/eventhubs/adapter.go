package eventhubs

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"

	"github.com/chris-alexander-pop/system-design-library/pkg/streaming"
)

// Ensure Adapter implements streaming.Client.
var _ streaming.Client = (*Adapter)(nil)

// Adapter provides a streaming.Client backed by Azure Event Hubs.
// The producer is bound to a single event hub at construction time.
type Adapter struct {
	client  *azeventhubs.ProducerClient
	hubName string
}

// New creates a new Event Hubs streaming adapter for the given namespace and hub.
func New(namespace, eventHub string) (*Adapter, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, streaming.ErrInvalidConfig("azure credential", err)
	}

	client, err := azeventhubs.NewProducerClient(namespace+".servicebus.windows.net", eventHub, cred, nil)
	if err != nil {
		return nil, streaming.ErrInvalidConfig("event hubs producer", err)
	}

	return &Adapter{client: client, hubName: eventHub}, nil
}

// PutRecord sends a single event. streamName must match the hub bound at New,
// or be empty (in which case the bound hub is used).
func (a *Adapter) PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error {
	if streamName != "" && streamName != a.hubName {
		return streaming.ErrStreamNotFound(streamName, nil)
	}

	batch, err := a.client.NewEventDataBatch(ctx, &azeventhubs.EventDataBatchOptions{
		PartitionKey: &partitionKey,
	})
	if err != nil {
		return streaming.ErrPutFailed(a.hubName, err)
	}

	if err := batch.AddEventData(&azeventhubs.EventData{Body: data}, nil); err != nil {
		return streaming.ErrPutFailed(a.hubName, err)
	}

	if err := a.client.SendEventDataBatch(ctx, batch, nil); err != nil {
		return streaming.ErrPutFailed(a.hubName, err)
	}
	return nil
}

// Close closes the Event Hubs producer client.
func (a *Adapter) Close() error {
	return a.client.Close(context.Background())
}
