package llm

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	options := ApplyOptions(opts...)

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
		telemetry.RecordError(span, err)
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

func (i *InstrumentedClient) StreamChat(ctx context.Context, messages []Message, opts ...GenerateOption) (<-chan GenerationChunk, error) {
	options := ApplyOptions(opts...)

	ctx, span := i.tracer.Start(ctx, "llm.StreamChat", trace.WithAttributes(
		attribute.String("llm.model", options.Model),
		attribute.Int("llm.message_count", len(messages)),
	))

	logger.L().InfoContext(ctx, "LLM stream chat request",
		"model", options.Model,
		"messages", len(messages),
	)

	upstream, err := i.next.StreamChat(ctx, messages, opts...)
	if err != nil {
		telemetry.RecordError(span, err)
		span.End()
		logger.L().ErrorContext(ctx, "LLM stream chat failed", "error", err)
		return nil, err
	}

	out := make(chan GenerationChunk)
	go func() {
		defer close(out)
		defer span.End()
		chunks := 0
		for chunk := range upstream {
			chunks++
			if chunk.Err != nil {
				telemetry.RecordError(span, chunk.Err)
			}
			if chunk.FinishReason != "" {
				span.SetAttributes(attribute.String("llm.finish_reason", chunk.FinishReason))
			}
			select {
			case out <- chunk:
			case <-ctx.Done():
				telemetry.RecordError(span, ctx.Err())
				return
			}
		}
		span.SetAttributes(attribute.Int("llm.chunk_count", chunks))
		logger.L().InfoContext(ctx, "LLM stream chat completed", "chunks", chunks)
	}()
	return out, nil
}

var _ Client = (*InstrumentedClient)(nil)
