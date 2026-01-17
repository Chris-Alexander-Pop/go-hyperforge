package kinesis

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
)

type Adapter struct {
	client *kinesis.Client
}

func New(ctx context.Context) (*Adapter, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		client: kinesis.NewFromConfig(cfg),
	}, nil
}

func (a *Adapter) PutRecord(ctx context.Context, streamName, partitionKey string, data []byte) error {
	_, err := a.client.PutRecord(ctx, &kinesis.PutRecordInput{
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
		Data:         data,
	})
	return err
}

func (a *Adapter) Close() error {
	return nil
}
