package gcp

import (
	"context"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

type Adapter struct {
	client *secretmanager.Client
}

func New(ctx context.Context) (*Adapter, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) GetSecret(ctx context.Context, key string) (string, error) {
	// Key format: projects/*/secrets/*/versions/*
	// If user passes short name, adapter might need to know project ID.
	// For now assume full resource name in 'key'.

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: key,
	}

	result, err := a.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", err
	}

	return string(result.Payload.Data), nil
}

func (a *Adapter) Close() error {
	return a.client.Close()
}
