// Package genai is a thin annotator on top of bolt that emits the
// OpenTelemetry GenAI semantic-convention field names for LLM calls
// and tool/function invocations.
//
// Field names track the OTel GenAI semconv (gen_ai.*). Keeping the
// schema in a separate go.mod lets this package move at semconv speed
// without churn in the v1 bolt core, and keeps the bolt core free of
// AI-specific concerns.
//
// This package is intentionally minimal:
//
//   - It does NOT bundle a tokenizer. Counts must come from the
//     LLM provider's response (where they're authoritative).
//   - It does NOT bundle a pricing table. Cost attribution belongs
//     in the OTel collector or a downstream tool (Langfuse, Phoenix,
//     Braintrust) where prices stay current.
//   - It does NOT do PII redaction. Use a [bolt.EventHook] for that.
//
// Field names are compatible with Langfuse, Phoenix, and Braintrust
// ingestion that respects the OTel GenAI semconv. Zero-valued fields
// in the input structs are skipped so callers only pay for what they
// populate.
//
// Example:
//
//	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
//	genai.Call(logger.Info(), genai.CallInfo{
//	    System:        "openai",
//	    Operation:     "chat",
//	    RequestModel:  "gpt-4o",
//	    ResponseModel: "gpt-4o-2024-08-06",
//	    InputTokens:   142,
//	    OutputTokens:  87,
//	    Latency:       450 * time.Millisecond,
//	    ResponseID:    "chatcmpl-…",
//	    FinishReason:  "stop",
//	}).Msg("chat call complete")
package genai

import (
	"time"

	"github.com/felixgeelhaar/bolt"
)

// OTel GenAI semantic-convention field names. Pinned here so that future
// updates to the semconv land in a single file. See:
//
//	https://opentelemetry.io/docs/specs/semconv/gen-ai/
const (
	// System and operation
	keySystem    = "gen_ai.system"
	keyOperation = "gen_ai.operation.name"

	// Request fields
	keyRequestModel       = "gen_ai.request.model"
	keyRequestMaxTokens   = "gen_ai.request.max_tokens" // #nosec G101 -- OTel semconv field name, not a credential
	keyRequestTemperature = "gen_ai.request.temperature"
	keyRequestTopP        = "gen_ai.request.top_p"

	// Response fields
	keyResponseModel        = "gen_ai.response.model"
	keyResponseID           = "gen_ai.response.id"
	keyResponseFinishReason = "gen_ai.response.finish_reasons"

	// Usage fields
	keyInputTokens  = "gen_ai.usage.input_tokens"  // #nosec G101 -- OTel semconv field name, not a credential
	keyOutputTokens = "gen_ai.usage.output_tokens" // #nosec G101 -- OTel semconv field name, not a credential
	keyTotalTokens  = "gen_ai.usage.total_tokens"  // #nosec G101 -- OTel semconv field name, not a credential

	// Timing
	keyDuration = "gen_ai.client.operation.duration"

	// Tool / function call fields
	keyToolName     = "gen_ai.tool.name"
	keyToolCallID   = "gen_ai.tool.call.id"
	keyToolArgsLen  = "gen_ai.tool.call.arguments.length"
	keyToolResLen   = "gen_ai.tool.call.result.length"
	keyToolDuration = "gen_ai.tool.call.duration"
)

// CallInfo describes a single LLM API call. Zero-valued fields are
// skipped during encoding.
type CallInfo struct {
	// System identifies the provider (e.g. "openai", "anthropic", "google").
	System string

	// Operation names the API operation (e.g. "chat", "generate_content",
	// "embeddings").
	Operation string

	// RequestModel is the model name requested by the caller.
	RequestModel string

	// ResponseModel is the model that actually served the response.
	// Often equal to RequestModel, but providers may resolve aliases.
	ResponseModel string

	// MaxTokens is the request's max_tokens parameter (0 == unset).
	MaxTokens int

	// Temperature is the request's temperature parameter (NaN == unset
	// — call sites that explicitly want temperature=0 should pass 0).
	Temperature float64

	// TopP is the request's top_p parameter. See Temperature note.
	TopP float64

	// InputTokens is the number of tokens in the prompt as reported by
	// the provider's response.
	InputTokens int64

	// OutputTokens is the number of tokens in the completion as
	// reported by the provider's response.
	OutputTokens int64

	// Latency is the wall-clock duration of the API call.
	Latency time.Duration

	// ResponseID is the provider's response identifier (e.g. OpenAI's
	// chatcmpl-… id), useful for cross-referencing with provider logs.
	ResponseID string

	// FinishReason names why generation stopped (e.g. "stop", "length",
	// "tool_calls"). Encoded as a single-element JSON array under
	// gen_ai.response.finish_reasons (the semconv field is a list to
	// accommodate multi-choice responses; this helper normalises the
	// common single-choice case).
	FinishReason string
}

// Call adds OTel GenAI semconv fields for a single LLM API call to the
// event and returns it for chaining. The caller terminates the chain
// with Msg or Send.
func Call(e *bolt.Event, c CallInfo) *bolt.Event {
	if c.System != "" {
		e = e.Str(keySystem, c.System)
	}
	if c.Operation != "" {
		e = e.Str(keyOperation, c.Operation)
	}
	if c.RequestModel != "" {
		e = e.Str(keyRequestModel, c.RequestModel)
	}
	if c.ResponseModel != "" && c.ResponseModel != c.RequestModel {
		e = e.Str(keyResponseModel, c.ResponseModel)
	}
	if c.MaxTokens > 0 {
		e = e.Int(keyRequestMaxTokens, c.MaxTokens)
	}
	if c.Temperature != 0 {
		e = e.Float64(keyRequestTemperature, c.Temperature)
	}
	if c.TopP != 0 {
		e = e.Float64(keyRequestTopP, c.TopP)
	}
	if c.InputTokens > 0 {
		e = e.Int64(keyInputTokens, c.InputTokens)
	}
	if c.OutputTokens > 0 {
		e = e.Int64(keyOutputTokens, c.OutputTokens)
	}
	if c.InputTokens > 0 || c.OutputTokens > 0 {
		e = e.Int64(keyTotalTokens, c.InputTokens+c.OutputTokens)
	}
	if c.Latency > 0 {
		e = e.Dur(keyDuration, c.Latency)
	}
	if c.ResponseID != "" {
		e = e.Str(keyResponseID, c.ResponseID)
	}
	if c.FinishReason != "" {
		// gen_ai.response.finish_reasons is a list per semconv; we emit
		// a one-element list (the common single-choice case).
		e = e.Strs(keyResponseFinishReason, []string{c.FinishReason})
	}
	return e
}

// ToolCallInfo describes a tool / function invocation made by an
// LLM-backed agent. Zero-valued fields are skipped during encoding.
type ToolCallInfo struct {
	// Name of the tool/function (e.g. "search_web", "execute_query").
	Name string

	// CallID is the provider-assigned identifier for this tool call,
	// useful for correlating with the provider's tool-call response.
	CallID string

	// ArgsLength is the byte size of the arguments payload. We log the
	// length, not the content, so PII in the arguments isn't dumped to
	// logs by default. Wrap with a redaction hook if you want the body.
	ArgsLength int

	// ResultLength is the byte size of the tool's result payload. Same
	// reasoning as ArgsLength.
	ResultLength int

	// Duration is the wall-clock duration of the tool execution.
	Duration time.Duration
}

// ToolCall adds OTel GenAI tool-call semconv fields to the event and
// returns it for chaining.
func ToolCall(e *bolt.Event, t ToolCallInfo) *bolt.Event {
	if t.Name != "" {
		e = e.Str(keyToolName, t.Name)
	}
	if t.CallID != "" {
		e = e.Str(keyToolCallID, t.CallID)
	}
	if t.ArgsLength > 0 {
		e = e.Int(keyToolArgsLen, t.ArgsLength)
	}
	if t.ResultLength > 0 {
		e = e.Int(keyToolResLen, t.ResultLength)
	}
	if t.Duration > 0 {
		e = e.Dur(keyToolDuration, t.Duration)
	}
	return e
}
