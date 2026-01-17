package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Adapter struct {
	client *secretsmanager.Client
}

func New(ctx context.Context) (*Adapter, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

func (a *Adapter) GetSecret(ctx context.Context, key string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}

	result, err := a.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", err
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	// Binary secret handling omitted for now (return empty or error?)
	return "", nil
}

func (a *Adapter) Close() error {
	return nil
}
