/*
Package ai provides artificial intelligence and machine learning capabilities.

This package organizes AI functionality into the following subdomains:

  - genai: Generative AI (LLMs, image generation, agents, gateway, prompts)
  - ml: Machine Learning (training, inference, feature stores)
  - nlp: Natural Language Processing (embeddings, RAG)
  - perception: Computer vision, speech, OCR

# LLM Chat

Use genai/llm for conversational Chat and StreamChat (not a separate Generate API):

	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
	resp, err := client.Chat(ctx, messages)
	chunks, err := client.StreamChat(ctx, messages)

# Multi-provider gateway

Use genai/gateway to try providers in order with fallback:

	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/gateway"
	r, err := gateway.New(primary, fallback)

# Prompt templates

Use genai/prompt for named/versioned template stubs ({{key}} substitution).

# Embedding Generation

For text embeddings, use nlp/embedding:

	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/nlp/embedding"
	vectors, err := embedder.Embed(ctx, texts)

Note: historical references to pkg/ai/llm are superseded by pkg/ai/genai/llm.
*/
package ai
