package pubsub

import (
	"context"

	"cloud.google.com/go/pubsub"
)

type Adapter struct {
	client *pubsub.Client
}

func New(ctx context.Context, projectID string) (*Adapter, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) PutRecord(ctx context.Context, topicName, partitionKey string, data []byte) error {
	// PubSub topic
	t := a.client.Topic(topicName)
	// Key used as ordering key
	res := t.Publish(ctx, &pubsub.Message{
		Data:        data,
		OrderingKey: partitionKey,
	})
	_, err := res.Get(ctx)
	return err
}

func (a *Adapter) Close() error {
	return a.client.Close()
}
