package evals_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/evals"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/adapters/memory"
)

func TestExactMatchRunner_GoldenSet(t *testing.T) {
	ctx := context.Background()
	client := memory.New().
		WithResponse("capital of france", "Paris").
		WithResponse("2+2", "4")

	set := evals.NewGoldenSet("basics",
		evals.Case{
			ID:       "france",
			Input:    []llm.Message{{Role: llm.RoleUser, Content: "What is the capital of France?"}},
			Expected: "Paris",
		},
		evals.Case{
			ID:       "math",
			Input:    []llm.Message{{Role: llm.RoleUser, Content: "What is 2+2?"}},
			Expected: "4",
		},
	)

	runner := evals.NewExactMatchRunner(client)
	report, err := runner.Run(ctx, set.Cases)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Passed != 2 || report.Failed != 0 {
		t.Fatalf("passed=%d failed=%d avg=%v results=%+v", report.Passed, report.Failed, report.Average, report.Results)
	}
	if report.Average != 1.0 {
		t.Fatalf("average=%v", report.Average)
	}
}

func TestLLMJudgeRunner_Memory(t *testing.T) {
	ctx := context.Background()
	candidate := memory.New().WithResponse("hello", "Hello there")
	// Judge unused for exact pre-judge path; still required non-nil.
	judge := memory.New().WithResponse("score", `{"score":0.9,"reason":"close enough"}`)

	runner := evals.NewLLMJudgeRunner(candidate, judge)
	report, err := runner.Run(ctx, []evals.Case{{
		ID:       "greet",
		Input:    []llm.Message{{Role: llm.RoleUser, Content: "say hello"}},
		Expected: "Hello there",
	}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !report.Results[0].Pass || report.Results[0].Score != 1.0 {
		t.Fatalf("result=%+v", report.Results[0])
	}
	if !strings.Contains(report.Results[0].Reason, "exact") {
		t.Fatalf("reason=%q", report.Results[0].Reason)
	}
}

func TestLLMJudgeRunner_MismatchUsesJudge(t *testing.T) {
	ctx := context.Background()
	candidate := memory.New().WithResponse("color", "blue")
	judge := memory.New().WithResponse("evaluation judge", `{"score":0.2,"reason":"wrong color"}`)

	runner := evals.NewLLMJudgeRunner(candidate, judge)
	runner.PassScore = 0.5
	report, err := runner.Run(ctx, []evals.Case{{
		ID:       "color",
		Input:    []llm.Message{{Role: llm.RoleUser, Content: "favorite color?"}},
		Expected: "red",
	}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Results[0].Pass {
		t.Fatalf("expected fail, got %+v", report.Results[0])
	}
	if report.Results[0].Score != 0.2 {
		t.Fatalf("score=%v reason=%q", report.Results[0].Score, report.Results[0].Reason)
	}
}
