package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/fraud"
	"github.com/google/uuid"
)

// Detector implements fraud.Detector using simple memory rules.
type Detector struct {
	// In a real memory adapter, we might store history to detect velocity attacks.
	// For now, we'll use simple static rules.
}

// New creates a new memory fraud detector.
func New() *Detector {
	return &Detector{}
}

func (d *Detector) Score(ctx context.Context, event fraud.UserEvent) (*fraud.Evaluation, error) {
	eval := &fraud.Evaluation{
		CheckID:   uuid.NewString(),
		Timestamp: time.Now(),
		Action:    "allow",
		RiskScore: 0.0,
		Metadata:  make(map[string]string),
	}

	// Simple simulated rules
	if event.Amount > 10000 {
		eval.RiskScore = 0.8
		eval.Action = "review"
		eval.Reasons = append(eval.Reasons, "high_amount")
	}

	if event.IPAddress == "1.2.3.4" { // Simulated bad IP
		eval.RiskScore = 1.0
		eval.Action = "block"
		eval.Reasons = append(eval.Reasons, "blacklisted_ip")
	}

	return eval, nil
}
