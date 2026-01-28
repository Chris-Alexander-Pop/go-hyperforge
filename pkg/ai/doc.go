/*
Package ai provides artificial intelligence and machine learning capabilities.

This package organizes AI functionality into the following subdomains:

  - genai: Generative AI (LLMs, image generation, agents)
  - ml: Machine Learning (training, inference, feature stores)
  - nlp: Natural Language Processing (embeddings, RAG)
  - perception: Computer vision, speech, OCR

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"

	client := openai.New("key")
	resp, err := client.Chat(ctx, messages)
*/
package ai
