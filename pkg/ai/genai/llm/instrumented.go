package llm

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedClient wraps an LLM Client with observability.
type InstrumentedClient struct {
	next   Client
	tracer trace.Tracer
}

// NewInstrumentedClient creates a new instrumented LLM client.
func NewInstrumentedClient(next Client) *InstrumentedClient {
	return &InstrumentedClient{
		next:   next,
		tracer: otel.Tracer("pkg/ai/genai/llm"),
	}
}

func (i *InstrumentedClient) Chat(ctx context.Context, messages []Message, opts ...GenerateOption) (*Generation, error) {
	options := GenerateOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	ctx, span := i.tracer.Start(ctx, "llm.Chat", trace.WithAttributes(
		attribute.String("llm.model", options.Model),
		attribute.Int("llm.message_count", len(messages)),
		attribute.Float64("llm.temperature", options.Temperature),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "LLM chat request",
		"model", options.Model,
		"messages", len(messages),
	)

	gen, err := i.next.Chat(ctx, messages, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "LLM chat failed", "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("llm.prompt_tokens", gen.Usage.PromptTokens),
		attribute.Int("llm.completion_tokens", gen.Usage.CompletionTokens),
		attribute.String("llm.finish_reason", gen.FinishReason),
	)

	logger.L().InfoContext(ctx, "LLM chat completed",
		"tokens", gen.Usage.TotalTokens,
		"finish_reason", gen.FinishReason,
	)

	return gen, nil
}

var _ Client = (*InstrumentedClient)(nil)
