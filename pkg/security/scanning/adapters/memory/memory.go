package memory

import (
	"context"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning"
)

// Scanner implements scanning.Scanner using simple memory rules.
type Scanner struct{}

// New creates a new memory scanner.
func New() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(ctx context.Context, resource scanning.Resource) (*scanning.Report, error) {
	report := &scanning.Report{
		ResourceID: resource.ID,
		ScannedAt:  time.Now(),
		Clean:      true,
	}

	// Simulated virus signature
	if strings.Contains(resource.Location, "eicar") || strings.Contains(resource.Location, "malware") {
		report.Clean = false
		report.Threats = append(report.Threats, "EICAR-Test-Signature")
	}

	return report, nil
}
