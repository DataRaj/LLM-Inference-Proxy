// Package models defines OpenAI-compatible response types for the proxy.
// Both the non-streaming (full JSON) and streaming (SSE) shapes are represented.
package models

// ChatCompletionResponse is the full, non-streaming response body from an
// upstream backend. Returned as-is to the caller when stream=false.
//
// Reference: https://platform.openai.com/docs/api-reference/chat/object
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a single completion candidate.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage reports token consumption for billing / observability.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Streaming (SSE) types
// ─────────────────────────────────────────────────────────────────────────────

// ChatCompletionChunk is a single server-sent event chunk emitted by an upstream
// when stream=true. The proxy forwards these verbatim to the client.
//
// Reference: https://platform.openai.com/docs/api-reference/chat/streaming
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

// ChunkChoice is the per-chunk choice delta.
type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"` // pointer — null until last chunk
}

// ChunkDelta carries the incremental content in a streaming chunk.
type ChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// SSEEvent is a parsed server-sent event as forwarded from upstream.
// Used internally when the proxy reads the upstream SSE stream and
// re-emits it to the client.
type SSEEvent struct {
	// Data is the raw JSON payload from "data: <payload>" lines.
	// "[DONE]" signals stream termination.
	Data string
}
