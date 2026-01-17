package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type Adapter struct {
	client *azsecrets.Client
}

func New(vaultURL string) (*Adapter, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, err
	}

	return &Adapter{client: client}, nil
}

func (a *Adapter) GetSecret(ctx context.Context, key string) (string, error) {
	// Azure Key Vault Secret names cannot have special chars probably?
	// SDK call:
	resp, err := a.client.GetSecret(ctx, key, "", nil)
	if err != nil {
		return "", err
	}

	return *resp.Value, nil
}

func (a *Adapter) Close() error {
	return nil
}
