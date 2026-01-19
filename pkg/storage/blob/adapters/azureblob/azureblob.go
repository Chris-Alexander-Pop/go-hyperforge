package azureblob

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type Adapter struct {
	client *azblob.Client
}

func New(accountName string) (*Adapter, error) {
	// Construct URL
	url := "https://" + accountName + ".blob.core.windows.net/"

	// DefaultCreds
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	client, err := azblob.NewClient(url, cred, nil)
	if err != nil {
		return nil, err
	}

	return &Adapter{client: client}, nil
}

func (a *Adapter) Put(ctx context.Context, container, blob string, data []byte) error {
	_, err := a.client.UploadBuffer(ctx, container, blob, data, nil)
	return err
}

func (a *Adapter) Get(ctx context.Context, container, blob string) ([]byte, error) {
	resp, err := a.client.DownloadStream(ctx, container, blob, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (a *Adapter) List(ctx context.Context, container string) ([]string, error) {
	var results []string
	pager := a.client.NewListBlobsFlatPager(container, nil)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, blob := range resp.Segment.BlobItems {
			results = append(results, *blob.Name)
		}
	}
	return results, nil
}

func (a *Adapter) Delete(ctx context.Context, container, blob string) error {
	_, err := a.client.DeleteBlob(ctx, container, blob, nil)
	return err
}

func (a *Adapter) Close() error {
	return nil
}
