// Package speech provides interfaces and adapters for Speech-to-Text (STT) and Text-to-Speech (TTS).
//
// Supported backends:
//   - Memory: in-memory mock for testing (adapters/memory)
//   - OpenAI: Whisper transcription + TTS (adapters/openai)
//   - AWS: Polly TTS + Transcribe STT via injectable SDK/HTTP (adapters/aws)
//   - Google: Cloud Speech-to-Text / Text-to-Speech thin HTTP client (adapters/google)
//
// Basic usage:
//
//	client := memory.New()
//	audio, err := client.TextToSpeech(ctx, "Hello world", speech.FormatMP3)
package speech
