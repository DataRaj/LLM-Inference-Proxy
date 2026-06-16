// Package models defines OpenAI-compatible request types for the proxy.
// These structs are used for JSON decoding on the inbound path and
// forwarded (re-encoded) to upstream backends.
package models

// ChatCompletionRequest mirrors the OpenAI Chat Completion request body.
// Only fields used by the proxy for routing and validation are strongly-typed;
// the remainder of the body is forwarded verbatim via RawBody.
//
// Reference: https://platform.openai.com/docs/api-reference/chat/create
type ChatCompletionRequest struct {
	// Model is the model identifier used for backend routing (e.g. "gpt-4o").
	Model string `json:"model"`

	// Messages is the conversation history.
	Messages []Message `json:"messages"`

	// Stream requests server-sent event streaming when true.
	Stream bool `json:"stream,omitempty"`

	// MaxTokens limits the number of tokens in the completion.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0–2.0).
	Temperature float64 `json:"temperature,omitempty"`

	// TopP is an alternative to temperature for nucleus sampling.
	TopP float64 `json:"top_p,omitempty"`

	// N specifies how many completion choices to generate.
	N int `json:"n,omitempty"`

	// Stop is a list of sequences where the API will stop generating further tokens.
	Stop []string `json:"stop,omitempty"`

	// User is an end-user identifier forwarded to the upstream for abuse tracking.
	User string `json:"user,omitempty"`
}

// Message represents a single turn in a conversation.
type Message struct {
	// Role is one of "system", "user", "assistant", or "tool".
	Role string `json:"role"`

	// Content is the text of the message.
	// May be a string or a structured content array; we accept string here.
	Content string `json:"content"`

	// Name is an optional name for the participant.
	Name string `json:"name,omitempty"`
}
