package awsiot

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iot"
	"github.com/aws/aws-sdk-go-v2/service/iot/types"
	"github.com/aws/aws-sdk-go-v2/service/iotdataplane"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config holds AWS IoT configuration.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // IoT data endpoint
}

// Client provides AWS IoT Core operations.
type Client struct {
	iot      *iot.Client
	data     *iotdataplane.Client
	config   Config
	endpoint string
}

// New creates a new AWS IoT client.
func New(cfg Config) (*Client, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to load AWS config", err)
	}

	iotClient := iot.NewFromConfig(awsCfg)

	// Get endpoint if not provided
	endpoint := cfg.Endpoint
	if endpoint == "" {
		resp, err := iotClient.DescribeEndpoint(context.Background(), &iot.DescribeEndpointInput{
			EndpointType: aws.String("iot:Data-ATS"),
		})
		if err != nil {
			return nil, pkgerrors.Internal("failed to get IoT endpoint", err)
		}
		endpoint = *resp.EndpointAddress
	}

	dataClient := iotdataplane.NewFromConfig(awsCfg, func(o *iotdataplane.Options) {
		o.BaseEndpoint = aws.String("https://" + endpoint)
	})

	return &Client{
		iot:      iotClient,
		data:     dataClient,
		config:   cfg,
		endpoint: endpoint,
	}, nil
}

// Thing represents an IoT thing.
type Thing struct {
	Name       string
	ARN        string
	ThingType  string
	Attributes map[string]string
	Version    int64
}

// CreateThing creates a new IoT thing.
func (c *Client) CreateThing(ctx context.Context, name string, attributes map[string]string) (*Thing, error) {
	input := &iot.CreateThingInput{
		ThingName: aws.String(name),
	}
	if len(attributes) > 0 {
		input.AttributePayload = &types.AttributePayload{
			Attributes: attributes,
		}
	}

	output, err := c.iot.CreateThing(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create thing", err)
	}

	return &Thing{
		Name:       *output.ThingName,
		ARN:        *output.ThingArn,
		Attributes: attributes,
	}, nil
}

// GetThing retrieves a thing by name.
func (c *Client) GetThing(ctx context.Context, name string) (*Thing, error) {
	output, err := c.iot.DescribeThing(ctx, &iot.DescribeThingInput{
		ThingName: aws.String(name),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("thing not found", err)
	}

	return &Thing{
		Name:       *output.ThingName,
		ARN:        *output.ThingArn,
		ThingType:  aws.ToString(output.ThingTypeName),
		Attributes: output.Attributes,
		Version:    output.Version,
	}, nil
}

// ListThings returns all things.
func (c *Client) ListThings(ctx context.Context) ([]*Thing, error) {
	output, err := c.iot.ListThings(ctx, &iot.ListThingsInput{})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list things", err)
	}

	things := make([]*Thing, len(output.Things))
	for i, t := range output.Things {
		things[i] = &Thing{
			Name:       *t.ThingName,
			ARN:        *t.ThingArn,
			ThingType:  aws.ToString(t.ThingTypeName),
			Attributes: t.Attributes,
		}
	}

	return things, nil
}

// DeleteThing removes a thing.
func (c *Client) DeleteThing(ctx context.Context, name string) error {
	_, err := c.iot.DeleteThing(ctx, &iot.DeleteThingInput{
		ThingName: aws.String(name),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete thing", err)
	}
	return nil
}

// Publish sends a message to an IoT topic.
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	_, err := c.data.Publish(ctx, &iotdataplane.PublishInput{
		Topic:   aws.String(topic),
		Payload: payload,
	})
	if err != nil {
		return pkgerrors.Internal("failed to publish message", err)
	}
	return nil
}

// PublishJSON publishes a JSON message.
func (c *Client) PublishJSON(ctx context.Context, topic string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return pkgerrors.Internal("failed to marshal JSON", err)
	}
	return c.Publish(ctx, topic, payload)
}

// Shadow represents a device shadow.
type Shadow struct {
	State     ShadowState            `json:"state"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Version   int                    `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
}

// ShadowState contains reported and desired states.
type ShadowState struct {
	Reported map[string]interface{} `json:"reported,omitempty"`
	Desired  map[string]interface{} `json:"desired,omitempty"`
	Delta    map[string]interface{} `json:"delta,omitempty"`
}

// GetShadow retrieves a thing's shadow.
func (c *Client) GetShadow(ctx context.Context, thingName string) (*Shadow, error) {
	output, err := c.data.GetThingShadow(ctx, &iotdataplane.GetThingShadowInput{
		ThingName: aws.String(thingName),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("shadow not found", err)
	}

	var shadow Shadow
	if err := json.Unmarshal(output.Payload, &shadow); err != nil {
		return nil, pkgerrors.Internal("failed to parse shadow", err)
	}

	return &shadow, nil
}

// UpdateShadow updates a thing's shadow.
func (c *Client) UpdateShadow(ctx context.Context, thingName string, reported, desired map[string]interface{}) error {
	state := map[string]interface{}{
		"state": map[string]interface{}{},
	}
	if reported != nil {
		state["state"].(map[string]interface{})["reported"] = reported
	}
	if desired != nil {
		state["state"].(map[string]interface{})["desired"] = desired
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return pkgerrors.Internal("failed to marshal shadow", err)
	}

	_, err = c.data.UpdateThingShadow(ctx, &iotdataplane.UpdateThingShadowInput{
		ThingName: aws.String(thingName),
		Payload:   payload,
	})
	if err != nil {
		return pkgerrors.Internal("failed to update shadow", err)
	}

	return nil
}

// DeleteShadow removes a thing's shadow.
func (c *Client) DeleteShadow(ctx context.Context, thingName string) error {
	_, err := c.data.DeleteThingShadow(ctx, &iotdataplane.DeleteThingShadowInput{
		ThingName: aws.String(thingName),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete shadow", err)
	}
	return nil
}

// GetEndpoint returns the IoT data endpoint.
func (c *Client) GetEndpoint() string {
	return c.endpoint
}
