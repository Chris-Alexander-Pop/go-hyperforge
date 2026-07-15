package kinesis

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/streaming"
)

// Ensure Adapter implements streaming.Client.
var _ streaming.Client = (*Adapter)(nil)

// Adapter provides a streaming.Client backed by AWS Kinesis.
type Adapter struct {
	client *kinesis.Client
}

// New creates a new Kinesis streaming adapter using the default AWS config chain.
func New(ctx context.Context) (*Adapter, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, streaming.ErrInvalidConfig("aws config", err)
	}
	return &Adapter{
		client: kinesis.NewFromConfig(cfg),
	}, nil
}

// PutRecord writes a single record to the named Kinesis stream.
func (a *Adapter) PutRecord(ctx context.Context, streamName, partitionKey string, data []byte) error {
	_, err := a.client.PutRecord(ctx, &kinesis.PutRecordInput{
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
		Data:         data,
	})
	if err != nil {
		return streaming.ErrPutFailed(streamName, err)
	}
	return nil
}

// PutRecords writes records sequentially via PutRecord.
func (a *Adapter) PutRecords(ctx context.Context, records []streaming.Record) error {
	for _, r := range records {
		if err := a.PutRecord(ctx, r.StreamName, r.PartitionKey, r.Data); err != nil {
			return err
		}
	}
	return nil
}

// Close is a no-op for the Kinesis SDK client.
func (a *Adapter) Close() error {
	return nil
}
