package eventhubs

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

type Adapter struct {
	client *azeventhubs.ProducerClient
}

func New(namespace, eventHub string) (*Adapter, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	client, err := azeventhubs.NewProducerClient(namespace+".servicebus.windows.net", eventHub, cred, nil)
	if err != nil {
		return nil, err
	}

	return &Adapter{client: client}, nil
}

func (a *Adapter) PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error {
	// Note: streamName argument might be redundant if client is already scoped to eventHub,
	// unless Client re-creates producers or manages multiple hubs.
	// For this adapter, we assume 'New' bound to specific Hub, so we ignore streamName or verify it match.

	batch, err := a.client.NewEventDataBatch(ctx, &azeventhubs.EventDataBatchOptions{
		PartitionKey: &partitionKey,
	})
	if err != nil {
		return err
	}

	if err := batch.AddEventData(&azeventhubs.EventData{Body: data}, nil); err != nil {
		return err
	}

	return a.client.SendEventDataBatch(ctx, batch, nil)
}

func (a *Adapter) Close() error {
	return a.client.Close(context.Background())
}
