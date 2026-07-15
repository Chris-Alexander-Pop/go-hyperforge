// Package evals provides LLM evaluation harnesses: golden sets and LLM-as-judge.
package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Case is one golden evaluation example.
type Case struct {
	ID       string                 `json:"id"`
	Input    []llm.Message          `json:"input"`
	Expected string                 `json:"expected"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CaseResult is the outcome of evaluating a single case.
type CaseResult struct {
	CaseID string  `json:"case_id"`
	Output string  `json:"output"`
	Pass   bool    `json:"pass"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason,omitempty"`
}

// Report aggregates evaluation results.
type Report struct {
	Results []CaseResult `json:"results"`
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
	Average float64      `json:"average_score"`
}

// EvalRunner runs a golden set against a candidate system.
type EvalRunner interface {
	Run(ctx context.Context, cases []Case) (*Report, error)
}

// GoldenSet is a named collection of evaluation cases.
type GoldenSet struct {
	Name  string `json:"name"`
	Cases []Case `json:"cases"`
}

// NewGoldenSet creates a golden set.
func NewGoldenSet(name string, cases ...Case) *GoldenSet {
	return &GoldenSet{Name: name, Cases: cases}
}

// ExactMatchRunner scores cases with exact (trimmed, case-insensitive) string match.
type ExactMatchRunner struct {
	Client llm.Client
}

// NewExactMatchRunner creates an exact-match EvalRunner.
func NewExactMatchRunner(client llm.Client) *ExactMatchRunner {
	return &ExactMatchRunner{Client: client}
}

func (r *ExactMatchRunner) Run(ctx context.Context, cases []Case) (*Report, error) {
	if r.Client == nil {
		return nil, llm.ErrNilClient
	}
	if len(cases) == 0 {
		return nil, errors.InvalidArgument("eval cases are required", nil)
	}
	report := &Report{Results: make([]CaseResult, 0, len(cases))}
	var sum float64
	for _, c := range cases {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		gen, err := r.Client.Chat(ctx, c.Input)
		if err != nil {
			return nil, errors.Wrap(err, "eval chat failed for case "+c.ID)
		}
		out := strings.TrimSpace(gen.Message.TextContent())
		pass := strings.EqualFold(out, strings.TrimSpace(c.Expected))
		score := 0.0
		reason := "mismatch"
		if pass {
			score = 1.0
			reason = "exact match"
			report.Passed++
		} else {
			report.Failed++
		}
		sum += score
		report.Results = append(report.Results, CaseResult{
			CaseID: c.ID,
			Output: out,
			Pass:   pass,
			Score:  score,
			Reason: reason,
		})
	}
	if len(report.Results) > 0 {
		report.Average = sum / float64(len(report.Results))
	}
	return report, nil
}

var _ EvalRunner = (*ExactMatchRunner)(nil)

// LLMJudgeRunner uses an LLM to score candidate outputs against expected answers.
type LLMJudgeRunner struct {
	Candidate llm.Client
	Judge     llm.Client
	PassScore float64 // minimum score [0,1] to pass; default 0.7
}

// NewLLMJudgeRunner creates an LLM-as-judge EvalRunner.
// When judge is nil, candidate is used for both generation and judging (memory-friendly).
func NewLLMJudgeRunner(candidate, judge llm.Client) *LLMJudgeRunner {
	if judge == nil {
		judge = candidate
	}
	return &LLMJudgeRunner{
		Candidate: candidate,
		Judge:     judge,
		PassScore: 0.7,
	}
}

func (r *LLMJudgeRunner) Run(ctx context.Context, cases []Case) (*Report, error) {
	if r.Candidate == nil || r.Judge == nil {
		return nil, llm.ErrNilClient
	}
	if len(cases) == 0 {
		return nil, errors.InvalidArgument("eval cases are required", nil)
	}
	passScore := r.PassScore
	if passScore <= 0 {
		passScore = 0.7
	}

	report := &Report{Results: make([]CaseResult, 0, len(cases))}
	var sum float64
	for _, c := range cases {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		gen, err := r.Candidate.Chat(ctx, c.Input)
		if err != nil {
			return nil, errors.Wrap(err, "eval candidate failed for case "+c.ID)
		}
		out := strings.TrimSpace(gen.Message.TextContent())
		score, reason, err := r.judge(ctx, c, out)
		if err != nil {
			return nil, err
		}
		pass := score >= passScore
		if pass {
			report.Passed++
		} else {
			report.Failed++
		}
		sum += score
		report.Results = append(report.Results, CaseResult{
			CaseID: c.ID,
			Output: out,
			Pass:   pass,
			Score:  score,
			Reason: reason,
		})
	}
	if len(report.Results) > 0 {
		report.Average = sum / float64(len(report.Results))
	}
	return report, nil
}

func (r *LLMJudgeRunner) judge(ctx context.Context, c Case, output string) (float64, string, error) {
	prompt := fmt.Sprintf(
		`You are an evaluation judge. Score how well the OUTPUT matches EXPECTED on a 0.0-1.0 scale.
Reply with JSON only: {"score":0.0,"reason":"..."}.

EXPECTED: %s
OUTPUT: %s`,
		c.Expected, output,
	)
	// Memory adapter: canned responses keyed by pattern. Prefer exact-match shortcut
	// when strings already match so tests stay deterministic without parsing.
	if strings.EqualFold(strings.TrimSpace(output), strings.TrimSpace(c.Expected)) {
		return 1.0, "exact match (pre-judge)", nil
	}

	gen, err := r.Judge.Chat(ctx, []llm.Message{
		{Role: llm.RoleSystem, Content: "You are a strict JSON-only evaluation judge."},
		{Role: llm.RoleUser, Content: prompt},
	})
	if err != nil {
		return 0, "", errors.Wrap(err, "judge chat failed for case "+c.ID)
	}
	return parseJudgeResponse(gen.Message.TextContent())
}

type judgePayload struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

func parseJudgeResponse(raw string) (float64, string, error) {
	raw = strings.TrimSpace(raw)
	// Try full JSON first.
	var payload judgePayload
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		return clampScore(payload.Score), payload.Reason, nil
	}
	// Extract JSON object substring if the model wrapped it.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(raw[start:end+1]), &payload); err == nil {
			return clampScore(payload.Score), payload.Reason, nil
		}
	}
	// Memory stub / non-JSON: treat as low score with raw reason.
	return 0.0, "unparseable judge response: " + truncate(raw, 120), nil
}

func clampScore(s float64) float64 {
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var _ EvalRunner = (*LLMJudgeRunner)(nil)
