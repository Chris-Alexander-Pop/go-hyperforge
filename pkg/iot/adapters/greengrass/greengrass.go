package greengrass

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/greengrassv2"
	"github.com/aws/aws-sdk-go-v2/service/greengrassv2/types"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config holds Greengrass configuration.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

// Client provides AWS Greengrass operations.
type Client struct {
	gg     *greengrassv2.Client
	config Config
}

// New creates a new Greengrass client.
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

	return &Client{
		gg:     greengrassv2.NewFromConfig(awsCfg),
		config: cfg,
	}, nil
}

// CoreDevice represents a Greengrass core device.
type CoreDevice struct {
	Name             string
	Status           string
	LastStatusUpdate string
	Platform         string
	Architecture     string
	CoreVersion      string
	Tags             map[string]string
}

// ListCoreDevices returns all core devices.
func (c *Client) ListCoreDevices(ctx context.Context) ([]*CoreDevice, error) {
	output, err := c.gg.ListCoreDevices(ctx, &greengrassv2.ListCoreDevicesInput{})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list core devices", err)
	}

	devices := make([]*CoreDevice, len(output.CoreDevices))
	for i, d := range output.CoreDevices {
		lastUpdate := ""
		if d.LastStatusUpdateTimestamp != nil {
			lastUpdate = d.LastStatusUpdateTimestamp.Format("2006-01-02T15:04:05Z")
		}
		devices[i] = &CoreDevice{
			Name:             *d.CoreDeviceThingName,
			Status:           string(d.Status),
			LastStatusUpdate: lastUpdate,
		}
	}

	return devices, nil
}

// GetCoreDevice retrieves a core device.
func (c *Client) GetCoreDevice(ctx context.Context, name string) (*CoreDevice, error) {
	output, err := c.gg.GetCoreDevice(ctx, &greengrassv2.GetCoreDeviceInput{
		CoreDeviceThingName: aws.String(name),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("core device not found", err)
	}

	return &CoreDevice{
		Name:         *output.CoreDeviceThingName,
		Status:       string(output.Status),
		Platform:     aws.ToString(output.Platform),
		Architecture: aws.ToString(output.Architecture),
		CoreVersion:  aws.ToString(output.CoreVersion),
		Tags:         output.Tags,
	}, nil
}

// Component represents a Greengrass component.
type Component struct {
	Name        string
	ARN         string
	Version     string
	Description string
	Publisher   string
	Status      string
}

// CreateComponent creates a Greengrass component.
func (c *Client) CreateComponent(ctx context.Context, name, version, recipe string) (*Component, error) {
	output, err := c.gg.CreateComponentVersion(ctx, &greengrassv2.CreateComponentVersionInput{
		InlineRecipe: []byte(recipe),
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create component", err)
	}

	return &Component{
		Name:    *output.ComponentName,
		ARN:     *output.Arn,
		Version: *output.ComponentVersion,
		Status:  string(output.Status.ComponentState),
	}, nil
}

// GetComponent retrieves a component.
func (c *Client) GetComponent(ctx context.Context, arn string) (*Component, error) {
	output, err := c.gg.GetComponent(ctx, &greengrassv2.GetComponentInput{
		Arn: aws.String(arn),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("component not found", err)
	}

	return &Component{
		ARN: arn,
		// Recipe contains the component details
		Description: string(output.Recipe),
	}, nil
}

// ListComponents returns all components.
func (c *Client) ListComponents(ctx context.Context) ([]*Component, error) {
	output, err := c.gg.ListComponents(ctx, &greengrassv2.ListComponentsInput{})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list components", err)
	}

	components := make([]*Component, len(output.Components))
	for i, comp := range output.Components {
		components[i] = &Component{
			Name: *comp.ComponentName,
			ARN:  *comp.Arn,
		}
		if comp.LatestVersion != nil {
			components[i].Version = *comp.LatestVersion.ComponentVersion
		}
	}

	return components, nil
}

// DeleteComponent removes a component.
func (c *Client) DeleteComponent(ctx context.Context, arn string) error {
	_, err := c.gg.DeleteComponent(ctx, &greengrassv2.DeleteComponentInput{
		Arn: aws.String(arn),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete component", err)
	}
	return nil
}

// Deployment represents a Greengrass deployment.
type Deployment struct {
	ID               string
	Name             string
	Target           string
	Status           string
	Components       map[string]string // component name -> version
	CreationTime     string
	IsLatestRevision bool
}

// CreateDeployment creates a deployment to a target.
func (c *Client) CreateDeployment(ctx context.Context, name, targetARN string, components map[string]string) (*Deployment, error) {
	compConfig := make(map[string]types.ComponentDeploymentSpecification)
	for compName, version := range components {
		compConfig[compName] = types.ComponentDeploymentSpecification{
			ComponentVersion: aws.String(version),
		}
	}

	output, err := c.gg.CreateDeployment(ctx, &greengrassv2.CreateDeploymentInput{
		DeploymentName: aws.String(name),
		TargetArn:      aws.String(targetARN),
		Components:     compConfig,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create deployment", err)
	}

	return &Deployment{
		ID:         *output.DeploymentId,
		Name:       name,
		Target:     targetARN,
		Components: components,
	}, nil
}

// GetDeployment retrieves a deployment.
func (c *Client) GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error) {
	output, err := c.gg.GetDeployment(ctx, &greengrassv2.GetDeploymentInput{
		DeploymentId: aws.String(deploymentID),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("deployment not found", err)
	}

	components := make(map[string]string)
	for name, spec := range output.Components {
		components[name] = aws.ToString(spec.ComponentVersion)
	}

	return &Deployment{
		ID:               *output.DeploymentId,
		Name:             aws.ToString(output.DeploymentName),
		Target:           *output.TargetArn,
		Status:           string(output.DeploymentStatus),
		Components:       components,
		IsLatestRevision: output.IsLatestForTarget,
	}, nil
}

// ListDeployments returns all deployments.
func (c *Client) ListDeployments(ctx context.Context) ([]*Deployment, error) {
	output, err := c.gg.ListDeployments(ctx, &greengrassv2.ListDeploymentsInput{})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list deployments", err)
	}

	deployments := make([]*Deployment, len(output.Deployments))
	for i, d := range output.Deployments {
		deployments[i] = &Deployment{
			ID:               *d.DeploymentId,
			Name:             aws.ToString(d.DeploymentName),
			Target:           *d.TargetArn,
			Status:           string(d.DeploymentStatus),
			IsLatestRevision: d.IsLatestForTarget,
		}
	}

	return deployments, nil
}

// CancelDeployment cancels a deployment.
func (c *Client) CancelDeployment(ctx context.Context, deploymentID string) error {
	_, err := c.gg.CancelDeployment(ctx, &greengrassv2.CancelDeploymentInput{
		DeploymentId: aws.String(deploymentID),
	})
	if err != nil {
		return pkgerrors.Internal("failed to cancel deployment", err)
	}
	return nil
}
