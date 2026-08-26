package conversation

import (
	"strings"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
)

func TestMessageUsageAccumulatorCombinesObservedAndUnobservedInput(t *testing.T) {
	accumulator := &messageUsageAccumulator{}

	firstCallMessages := []llm.Message{{Role: "user", Content: "hello"}}
	accumulator.beginCall(llm.GenerateInput{Messages: firstCallMessages})
	accumulator.addObservedUsage(llm.Usage{InputTokens: 12, OutputTokens: 3})

	if got := accumulator.interruptedInputTokens(); got != 12 {
		t.Fatalf("expected observed input tokens after first call, got %d", got)
	}

	secondCallMessages := []llm.Message{{Role: "tool", Content: "tool result"}}
	secondCallInput := llm.GenerateInput{Messages: secondCallMessages}
	secondCallEstimate := estimateGenerateInputTokens(secondCallInput)
	accumulator.beginCall(secondCallInput)
	accumulator.finishCall(false)

	want := int64(12) + secondCallEstimate
	if got := accumulator.interruptedInputTokens(); got != want {
		t.Fatalf("expected observed plus unobserved input tokens, got %d want %d", got, want)
	}
	if got := accumulator.effectiveInputTokens(0); got != want {
		t.Fatalf("expected effective input tokens to include unobserved estimate, got %d want %d", got, want)
	}
}

func TestMessageUsageAccumulatorManagedModeTrustsUpstreamOnly(t *testing.T) {
	accumulator := &messageUsageAccumulator{}
	accumulator.setManagedMode(true)

	callInput := llm.GenerateInput{Messages: []llm.Message{{Role: "user", Content: "full history that would otherwise be a large estimate"}}}
	accumulator.beginCall(callInput)
	accumulator.finishCall(false)

	if got := accumulator.effectiveInputTokens(50000); got != 0 {
		t.Fatalf("managed mode must not fall back to local estimate, got %d", got)
	}

	accumulator.beginCall(callInput)
	accumulator.addObservedUsage(llm.Usage{InputTokens: 6381, OutputTokens: 3})
	if got := accumulator.effectiveInputTokens(50000); got != 6381 {
		t.Fatalf("managed mode must use upstream observed usage, got %d want 6381", got)
	}
}

func TestIsManagedContextStrategy(t *testing.T) {
	cases := map[string]bool{
		"":                  false,
		"full_history":      false,
		"managed_bootstrap": true,
		"managed_delta":     true,
		"  managed_delta ":  true,
	}
	for strategy, want := range cases {
		if got := isManagedContextStrategy(strategy); got != want {
			t.Fatalf("isManagedContextStrategy(%q) = %v, want %v", strategy, got, want)
		}
	}
}

func TestResolveObservedOrHigherEstimatedTokensKeepsLargerEstimate(t *testing.T) {
	if got := resolveObservedOrHigherEstimatedTokens(40, 96); got != 96 {
		t.Fatalf("expected larger input estimate, got %d", got)
	}
	if got := resolveObservedOrHigherEstimatedOutputTokens(2, "hello world this is a longer streamed response"); got <= 2 {
		t.Fatalf("expected output estimate to cover partial observed usage, got %d", got)
	}
}

func TestResolveObservedOrEstimatedTokensPrefersObservedUsage(t *testing.T) {
	if got := resolveObservedOrEstimatedTokens(40, 96); got != 40 {
		t.Fatalf("expected successful usage path to prefer observed tokens, got %d", got)
	}
	if got := resolveObservedOrEstimatedOutputTokens(7, "hello world this is a longer streamed response"); got != 7 {
		t.Fatalf("expected successful output usage path to prefer observed tokens, got %d", got)
	}
}

func TestEstimateTokensCollapsesDataImageURLs(t *testing.T) {
	content := "![preview](data:image/png;base64," + strings.Repeat("A", 8000) + ")"

	got := estimateTokens(content)
	if got > 20 {
		t.Fatalf("expected data image payload to be collapsed before estimating tokens, got %d", got)
	}
}

func TestEstimateGenerateInputTokensIncludesInstructionsAndTools(t *testing.T) {
	input := llm.GenerateInput{
		Messages:     []llm.Message{{Role: "user", Content: "hello"}},
		Instructions: "answer tersely",
		Tools: []llm.ToolDefinition{{
			Name:        "lookup",
			Description: "Search docs",
			InputSchema: []byte(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		}},
	}

	messageOnly := estimatePromptTokens(input.Messages)
	withInputShape := estimateGenerateInputTokens(input)
	if withInputShape <= messageOnly {
		t.Fatalf("expected generate input estimate to include instructions and tools, got %d <= %d", withInputShape, messageOnly)
	}

	input.DisableTools = true
	withoutTools := estimateGenerateInputTokens(input)
	if withoutTools >= withInputShape {
		t.Fatalf("expected disabled tools to be excluded from estimate, got %d >= %d", withoutTools, withInputShape)
	}
}

func TestSendMessageBillingDurationSeconds(t *testing.T) {
	if got := sendMessageBillingDurationSeconds(&SendMessageResult{DurationSeconds: 5}, 1200); got != 5 {
		t.Fatalf("expected explicit duration seconds to win, got %d", got)
	}
	if got := sendMessageBillingDurationSeconds(&SendMessageResult{}, 1201); got != 2 {
		t.Fatalf("expected latency to be rounded up to seconds, got %d", got)
	}
	if got := sendMessageBillingDurationSeconds(&SendMessageResult{}, 0); got != 0 {
		t.Fatalf("expected empty duration for zero latency, got %d", got)
	}
}

func TestMediaDurationSecondsFromOptions(t *testing.T) {
	if got := mediaDurationSecondsFromOptions(map[string]interface{}{"durationSeconds": float64(5)}); got != 5 {
		t.Fatalf("expected numeric duration seconds, got %d", got)
	}
	if got := mediaDurationSecondsFromOptions(map[string]interface{}{"duration": "5.2s"}); got != 6 {
		t.Fatalf("expected string duration to round up, got %d", got)
	}
	if got := mediaDurationSecondsFromOptions(map[string]interface{}{"duration": "bad"}); got != 0 {
		t.Fatalf("expected invalid duration to be ignored, got %d", got)
	}
}
