// Package textract provides an AWS Textract OCR adapter skeleton.
package textract

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/ocr"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config holds Textract configuration.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

// Client implements ocr.OCRClient using AWS Textract.
type Client struct {
	client *textract.Client
}

// New creates a Textract OCR client.
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
		return nil, pkgerrors.Internal("failed to load AWS config for textract", err)
	}
	return &Client{client: textract.NewFromConfig(awsCfg)}, nil
}

// DetectText extracts text from a document via DetectDocumentText.
func (c *Client) DetectText(ctx context.Context, document ocr.Document) (*ocr.TextResult, error) {
	if len(document.Content) == 0 && document.URI == "" {
		return nil, pkgerrors.InvalidArgument("document content or URI is required", nil)
	}

	input := &textract.DetectDocumentTextInput{
		Document: &types.Document{},
	}
	if len(document.Content) > 0 {
		input.Document.Bytes = document.Content
	} else {
		// S3 object URI form: s3://bucket/key
		bucket, key, err := parseS3URI(document.URI)
		if err != nil {
			return nil, err
		}
		input.Document.S3Object = &types.S3Object{
			Bucket: aws.String(bucket),
			Name:   aws.String(key),
		}
	}

	out, err := c.client.DetectDocumentText(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("textract DetectDocumentText failed", err)
	}

	result := &ocr.TextResult{Pages: []ocr.Page{{Number: 1}}}
	var textParts []byte
	for i, block := range out.Blocks {
		if block.BlockType != types.BlockTypeLine && block.BlockType != types.BlockTypeWord {
			continue
		}
		txt := ""
		if block.Text != nil {
			txt = *block.Text
		}
		conf := 0.0
		if block.Confidence != nil {
			conf = float64(*block.Confidence) / 100.0
		}
		bb := []float64{}
		if block.Geometry != nil && block.Geometry.BoundingBox != nil {
			b := block.Geometry.BoundingBox
			bb = []float64{
				float64(b.Left),
				float64(b.Top),
				float64(b.Width),
				float64(b.Height),
			}
		}
		result.Pages[0].Blocks = append(result.Pages[0].Blocks, ocr.Block{
			ID:          fmt.Sprintf("block-%d", i),
			Text:        txt,
			Type:        string(block.BlockType),
			Confidence:  conf,
			BoundingBox: bb,
		})
		if block.BlockType == types.BlockTypeLine && txt != "" {
			if len(textParts) > 0 {
				textParts = append(textParts, '\n')
			}
			textParts = append(textParts, txt...)
		}
	}
	result.Text = string(textParts)
	return result, nil
}

func parseS3URI(uri string) (bucket, key string, err error) {
	const prefix = "s3://"
	if len(uri) < len(prefix) || uri[:len(prefix)] != prefix {
		return "", "", pkgerrors.InvalidArgument("textract URI must be s3://bucket/key when content is empty", nil)
	}
	rest := uri[len(prefix):]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			if i == 0 || i == len(rest)-1 {
				return "", "", pkgerrors.InvalidArgument("invalid s3 URI", nil)
			}
			return rest[:i], rest[i+1:], nil
		}
	}
	return "", "", pkgerrors.InvalidArgument("invalid s3 URI", nil)
}

var _ ocr.OCRClient = (*Client)(nil)
