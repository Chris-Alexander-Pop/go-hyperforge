// Package guardduty implements scanning.Scanner by listing AWS GuardDuty findings.
//
// Scan maps a Resource to detector findings filtered by resource location/ID.
// Inject FindingsAPI via NewFromAPI for tests; New builds the SDK client.
package guardduty

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning"
)

// FindingsAPI is the GuardDuty surface used by this adapter.
// *guardduty.Client from aws-sdk-go-v2 satisfies this interface.
type FindingsAPI interface {
	ListFindings(ctx context.Context, params *guardduty.ListFindingsInput, optFns ...func(*guardduty.Options)) (*guardduty.ListFindingsOutput, error)
	GetFindings(ctx context.Context, params *guardduty.GetFindingsInput, optFns ...func(*guardduty.Options)) (*guardduty.GetFindingsOutput, error)
}

// Config configures the GuardDuty scanner adapter.
type Config struct {
	// Region is the AWS region (required for New).
	Region string `env:"AWS_REGION" env-default:"us-east-1"`

	// DetectorID is the GuardDuty detector to query (required).
	DetectorID string `env:"GUARDDUTY_DETECTOR_ID"`

	// AccessKeyID / SecretAccessKey are optional static credentials.
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`

	// Endpoint overrides the GuardDuty endpoint (LocalStack / tests).
	Endpoint string `env:"AWS_GUARDDUTY_ENDPOINT"`
}

// Scanner implements scanning.Scanner via AWS GuardDuty findings.
type Scanner struct {
	client     FindingsAPI
	detectorID string
}

// Ensure Scanner implements scanning.Scanner.
var _ scanning.Scanner = (*Scanner)(nil)

// NewFromAPI wraps an existing FindingsAPI (SDK client or test double).
func NewFromAPI(api FindingsAPI, detectorID string) (*Scanner, error) {
	if api == nil {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "guardduty api client is required", nil)
	}
	if detectorID == "" {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "guardduty detector id is required", nil)
	}
	return &Scanner{client: api, detectorID: detectorID}, nil
}

// New builds a Scanner from AWS SDK config.
func New(ctx context.Context, cfg Config) (*Scanner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "aws region is required", nil)
	}
	if cfg.DetectorID == "" {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "guardduty detector id is required", nil)
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.New(scanning.CodeUnavailable, "failed to load aws config", err)
	}

	client := guardduty.NewFromConfig(awsCfg, func(o *guardduty.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return NewFromAPI(client, cfg.DetectorID)
}

// Scan lists GuardDuty findings related to the resource and reports threats.
// Resource.Location (preferred) or Resource.ID is matched against finding titles/descriptions.
func (s *Scanner) Scan(ctx context.Context, resource scanning.Resource) (*scanning.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if resource.ID == "" && resource.Location == "" {
		return nil, scanning.ErrInvalidResource
	}

	needle := resource.Location
	if needle == "" {
		needle = resource.ID
	}

	listOut, err := s.client.ListFindings(ctx, &guardduty.ListFindingsInput{
		DetectorId: aws.String(s.detectorID),
		FindingCriteria: &types.FindingCriteria{
			Criterion: map[string]types.Condition{
				"service.archived": {Equals: []string{"false"}},
			},
		},
		MaxResults: aws.Int32(50),
	})
	if err != nil {
		return nil, pkgerrors.New(scanning.CodeUnavailable, "guardduty list findings failed", err)
	}

	report := &scanning.Report{
		ResourceID: resource.ID,
		Clean:      true,
		ScannedAt:  time.Now().UTC(),
	}
	if listOut == nil || len(listOut.FindingIds) == 0 {
		return report, nil
	}

	getOut, err := s.client.GetFindings(ctx, &guardduty.GetFindingsInput{
		DetectorId: aws.String(s.detectorID),
		FindingIds: listOut.FindingIds,
	})
	if err != nil {
		return nil, pkgerrors.New(scanning.CodeScanFailed, "guardduty get findings failed", err)
	}
	if getOut == nil {
		return report, nil
	}

	lowerNeedle := strings.ToLower(needle)
	for _, f := range getOut.Findings {
		title := aws.ToString(f.Title)
		desc := aws.ToString(f.Description)
		atype := aws.ToString(f.Type)
		blob := strings.ToLower(title + " " + desc + " " + atype)
		if lowerNeedle != "" && !strings.Contains(blob, lowerNeedle) && !resourceMatchesFinding(resource, f) {
			continue
		}
		threat := atype
		if threat == "" {
			threat = title
		}
		if threat == "" {
			threat = fmt.Sprintf("finding:%s", aws.ToString(f.Id))
		}
		report.Threats = append(report.Threats, threat)
	}
	if len(report.Threats) > 0 {
		report.Clean = false
	}
	return report, nil
}

func resourceMatchesFinding(resource scanning.Resource, f types.Finding) bool {
	if f.Resource == nil {
		return resource.Location == "" && resource.ID == ""
	}
	id := resource.ID
	loc := resource.Location
	if id == "" && loc == "" {
		return true
	}
	check := func(s *string) bool {
		if s == nil {
			return false
		}
		v := *s
		return (id != "" && strings.Contains(v, id)) || (loc != "" && strings.Contains(v, loc))
	}
	if f.Resource.InstanceDetails != nil && check(f.Resource.InstanceDetails.InstanceId) {
		return true
	}
	if f.Resource.S3BucketDetails != nil {
		for _, b := range f.Resource.S3BucketDetails {
			if check(b.Name) || check(b.Arn) {
				return true
			}
		}
	}
	return false
}
