package dynamodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/document"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// NOTE: This implementation assumes the AWS SDK v2 is available.

// Adapter implements the document.Interface for AWS DynamoDB.
type Adapter struct {
	client *dynamodb.Client
}

// New creates a new DynamoDB adapter.
func New(cfg document.Config) (*Adapter, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	// Support for local DynamoDB (e.g. Docker)
	var clientOpts []func(*dynamodb.Options)
	if cfg.Host != "" && cfg.Port != 0 {
		clientOpts = append(clientOpts, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port))
		})
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, errors.Internal("failed to load aws config", err)
	}

	return &Adapter{
		client: dynamodb.NewFromConfig(awsCfg, clientOpts...),
	}, nil
}

// Insert adds a new item to DynamoDB.
func (a *Adapter) Insert(ctx context.Context, collection string, doc document.Document) error {
	av, err := attributevalue.MarshalMap(doc)
	if err != nil {
		return errors.Internal("failed to marshal document", err)
	}

	_, err = a.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(collection),
		Item:      av,
	})
	if err != nil {
		return errors.Internal("failed to put item to dynamodb", err)
	}
	return nil
}

// Find searches for items in DynamoDB.
// Query map format: {"pk": "value", "sk": "value"} or simple filter.
// Note: This is a simplified Find implementation. Real DynamoDB queries are complex.
func (a *Adapter) Find(ctx context.Context, collection string, query map[string]interface{}) ([]document.Document, error) {
	// If PK is present, use GetItem or Query. Otherwise Scan (which is slow).
	// For this generic impl, we'll try to build a Scan with FilterExpression if not PK lookup.

	// NOTE: This specific implementation of Find is highly simplified and assumes equality checks.

	// Helper to handle Scan/Query construction omitted for brevity, focusing on Scan with Filter.
	// In production, this should prefer Query if PK/SK are identified.

	var filterExp *string
	var expAttrValues map[string]types.AttributeValue
	var expAttrNames map[string]string

	if len(query) > 0 {
		parts := []string{}
		expAttrValues = make(map[string]types.AttributeValue)
		expAttrNames = make(map[string]string)

		i := 0
		for k, v := range query {
			placeholder := fmt.Sprintf(":v%d", i)
			namePlaceholder := fmt.Sprintf("#n%d", i)

			parts = append(parts, fmt.Sprintf("%s = %s", namePlaceholder, placeholder))

			av, err := attributevalue.Marshal(v)
			if err != nil {
				return nil, errors.Internal("failed to marshal query value", err)
			}
			expAttrValues[placeholder] = av
			expAttrNames[namePlaceholder] = k
			i++
		}
		f := strings.Join(parts, " AND ")
		filterExp = &f
	}

	output, err := a.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:                 aws.String(collection),
		FilterExpression:          filterExp,
		ExpressionAttributeValues: expAttrValues,
		ExpressionAttributeNames:  expAttrNames,
	})
	if err != nil {
		return nil, errors.Internal("failed to scan dynamodb", err)
	}

	var docs []document.Document
	for _, item := range output.Items {
		var doc document.Document
		if err := attributevalue.UnmarshalMap(item, &doc); err != nil {
			return nil, errors.Internal("failed to unmarshal dynamodb item", err)
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// Update modifies an item.
func (a *Adapter) Update(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) error {
	// DynamoDB UpdateItem requires keys. We assume filter contains the Primary Key.
	key, err := attributevalue.MarshalMap(filter)
	if err != nil {
		return errors.Internal("failed to marshal key", err)
	}

	// Update Expression construction
	parts := []string{}
	expAttrValues := make(map[string]types.AttributeValue)
	expAttrNames := make(map[string]string)

	i := 0
	for k, v := range update {
		placeholder := fmt.Sprintf(":v%d", i)
		namePlaceholder := fmt.Sprintf("#n%d", i)

		parts = append(parts, fmt.Sprintf("%s = %s", namePlaceholder, placeholder))

		av, err := attributevalue.Marshal(v)
		if err != nil {
			return errors.Internal("failed to marshal update value", err)
		}
		expAttrValues[placeholder] = av
		expAttrNames[namePlaceholder] = k
		i++
	}

	updateExp := "SET " + strings.Join(parts, ", ")

	_, err = a.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(collection),
		Key:                       key,
		UpdateExpression:          aws.String(updateExp),
		ExpressionAttributeValues: expAttrValues,
		ExpressionAttributeNames:  expAttrNames,
	})
	if err != nil {
		return errors.Internal("failed to update dynamodb item", err)
	}

	return nil
}

// Delete removes an item.
func (a *Adapter) Delete(ctx context.Context, collection string, filter map[string]interface{}) error {
	// DynamoDB DeleteItem requires keys. We assume filter contains the Primary Key.
	key, err := attributevalue.MarshalMap(filter)
	if err != nil {
		return errors.Internal("failed to marshal key", err)
	}

	_, err = a.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(collection),
		Key:       key,
	})
	if err != nil {
		return errors.Internal("failed to delete dynamodb item", err)
	}
	return nil
}

// Close is a no-op for AWS Client.
func (a *Adapter) Close() error {
	return nil
}
