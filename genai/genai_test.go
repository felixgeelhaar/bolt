package genai_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bolt/genai"
)

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", raw, err)
	}
	return m
}

func TestCall_AllFields(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf))

	genai.Call(logger.Info(), genai.CallInfo{
		System:        "openai",
		Operation:     "chat",
		RequestModel:  "gpt-4o",
		ResponseModel: "gpt-4o-2024-08-06",
		MaxTokens:     1024,
		Temperature:   0.7,
		TopP:          0.9,
		InputTokens:   142,
		OutputTokens:  87,
		Latency:       450 * time.Millisecond,
		ResponseID:    "chatcmpl-abc123",
		FinishReason:  "stop",
	}).Msg("chat call done")

	got := decode(t, buf.Bytes())

	want := map[string]any{
		"gen_ai.system":                  "openai",
		"gen_ai.operation.name":          "chat",
		"gen_ai.request.model":           "gpt-4o",
		"gen_ai.response.model":          "gpt-4o-2024-08-06",
		"gen_ai.request.max_tokens":      float64(1024),
		"gen_ai.request.temperature":     0.7,
		"gen_ai.request.top_p":           0.9,
		"gen_ai.usage.input_tokens":      float64(142),
		"gen_ai.usage.output_tokens":     float64(87),
		"gen_ai.usage.total_tokens":      float64(229),
		"gen_ai.client.operation.duration": float64(450 * time.Millisecond),
		"gen_ai.response.id":             "chatcmpl-abc123",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("field %q: got %v, want %v", k, got[k], v)
		}
	}

	reasons, ok := got["gen_ai.response.finish_reasons"].([]any)
	if !ok {
		t.Fatalf("finish_reasons should be []string, got %T", got["gen_ai.response.finish_reasons"])
	}
	if len(reasons) != 1 || reasons[0] != "stop" {
		t.Errorf("finish_reasons = %v, want [stop]", reasons)
	}
}

func TestCall_ZeroValuesSkipped(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf))

	// Only system + tokens populated; everything else zero.
	genai.Call(logger.Info(), genai.CallInfo{
		System:       "anthropic",
		InputTokens:  10,
		OutputTokens: 5,
	}).Msg("partial")

	out := buf.String()
	if !strings.Contains(out, `"gen_ai.system":"anthropic"`) {
		t.Errorf("system field missing: %q", out)
	}
	if !strings.Contains(out, `"gen_ai.usage.total_tokens":15`) {
		t.Errorf("total_tokens (computed) missing: %q", out)
	}
	for _, banned := range []string{
		"gen_ai.operation.name",
		"gen_ai.request.model",
		"gen_ai.response.model",
		"gen_ai.request.max_tokens",
		"gen_ai.request.temperature",
		"gen_ai.request.top_p",
		"gen_ai.client.operation.duration",
		"gen_ai.response.id",
		"gen_ai.response.finish_reasons",
	} {
		if strings.Contains(out, banned) {
			t.Errorf("zero-valued field %q should have been skipped: %q", banned, out)
		}
	}
}

func TestCall_ResponseModelEqualToRequestModelSkipped(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf))

	genai.Call(logger.Info(), genai.CallInfo{
		RequestModel:  "gpt-4o",
		ResponseModel: "gpt-4o", // identical
	}).Msg("ok")

	out := buf.String()
	if !strings.Contains(out, `"gen_ai.request.model":"gpt-4o"`) {
		t.Errorf("request.model missing: %q", out)
	}
	if strings.Contains(out, `"gen_ai.response.model"`) {
		t.Errorf("response.model should be skipped when equal to request.model: %q", out)
	}
}

func TestCall_TotalTokensComputedFromInputOrOutputAlone(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf))

	// InputTokens only, no OutputTokens.
	genai.Call(logger.Info(), genai.CallInfo{InputTokens: 50}).Msg("input only")
	if !strings.Contains(buf.String(), `"gen_ai.usage.total_tokens":50`) {
		t.Errorf("total_tokens should equal input alone, got %q", buf.String())
	}

	buf.Reset()
	// OutputTokens only.
	genai.Call(logger.Info(), genai.CallInfo{OutputTokens: 30}).Msg("output only")
	if !strings.Contains(buf.String(), `"gen_ai.usage.total_tokens":30`) {
		t.Errorf("total_tokens should equal output alone, got %q", buf.String())
	}
}

func TestToolCall_AllFields(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf))

	genai.ToolCall(logger.Info(), genai.ToolCallInfo{
		Name:         "search_web",
		CallID:       "call_xyz",
		ArgsLength:   142,
		ResultLength: 8192,
		Duration:     1200 * time.Millisecond,
	}).Msg("tool call done")

	got := decode(t, buf.Bytes())
	want := map[string]any{
		"gen_ai.tool.name":                   "search_web",
		"gen_ai.tool.call.id":                "call_xyz",
		"gen_ai.tool.call.arguments.length":  float64(142),
		"gen_ai.tool.call.result.length":     float64(8192),
		"gen_ai.tool.call.duration":          float64(1200 * time.Millisecond),
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("field %q: got %v, want %v", k, got[k], v)
		}
	}
}

func TestCall_ChainsWithOtherFields(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf))

	// Pattern: build the event with bolt fields, then hand it to genai.Call
	// for the GenAI semconv block, then terminate with Msg.
	e := logger.Info().
		Str("user_id", "u-123").
		Str("tenant", "acme")
	genai.Call(e, genai.CallInfo{
		System:    "openai",
		Operation: "chat",
	}).Msg("done")

	out := buf.String()
	if !strings.Contains(out, `"user_id":"u-123"`) {
		t.Errorf("pre-genai field lost: %q", out)
	}
	if !strings.Contains(out, `"gen_ai.system":"openai"`) {
		t.Errorf("genai field missing: %q", out)
	}
}
