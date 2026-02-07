package otp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// RecoveryCodeConfig configures recovery code generation.
type RecoveryCodeConfig struct {
	// Count is the number of recovery codes to generate.
	Count int

	// Length is the length of each code in bytes (will be hex-encoded).
	Length int

	// GroupSize is the number of characters per group for readability.
	GroupSize int
}

// DefaultRecoveryCodeConfig returns sensible defaults.
func DefaultRecoveryCodeConfig() RecoveryCodeConfig {
	return RecoveryCodeConfig{
		Count:     10,
		Length:    8, // 16 hex characters
		GroupSize: 4,
	}
}

// RecoveryCodeManager manages recovery codes.
type RecoveryCodeManager struct {
	config RecoveryCodeConfig
}

// NewRecoveryCodeManager creates a new recovery code manager.
func NewRecoveryCodeManager(cfg RecoveryCodeConfig) *RecoveryCodeManager {
	return &RecoveryCodeManager{config: cfg}
}

// GenerateCodes generates a set of recovery codes.
// Returns both the formatted codes (for display) and the hashed codes (for storage).
func (m *RecoveryCodeManager) GenerateCodes() (displayCodes []string, hashedCodes []string, err error) {
	displayCodes = make([]string, m.config.Count)
	hashedCodes = make([]string, m.config.Count)

	for i := 0; i < m.config.Count; i++ {
		code := make([]byte, m.config.Length)
		if _, err := rand.Read(code); err != nil {
			return nil, nil, err
		}

		raw := hex.EncodeToString(code)
		displayCodes[i] = m.formatCode(raw)

		// Hash the raw code before returning it for storage
		hash := sha256.Sum256([]byte(raw))
		hashedCodes[i] = hex.EncodeToString(hash[:])
	}

	return displayCodes, hashedCodes, nil
}

// formatCode formats a code with groups for readability.
func (m *RecoveryCodeManager) formatCode(code string) string {
	if m.config.GroupSize <= 0 {
		return code
	}

	var groups []string
	for i := 0; i < len(code); i += m.config.GroupSize {
		end := i + m.config.GroupSize
		if end > len(code) {
			end = len(code)
		}
		groups = append(groups, code[i:end])
	}
	return strings.Join(groups, "-")
}

// NormalizeCode removes formatting from a code for comparison.
func (m *RecoveryCodeManager) NormalizeCode(code string) string {
	return strings.ReplaceAll(strings.ToLower(code), "-", "")
}

// RecoveryCodeSet is a thread-safe set of recovery codes.
// Each code can only be used once.
type RecoveryCodeSet struct {
	codes map[string]bool // code -> used
	mu    *concurrency.SmartRWMutex
}

// NewRecoveryCodeSet creates a new recovery code set.
func NewRecoveryCodeSet(hashedCodes []string) *RecoveryCodeSet {
	codes := make(map[string]bool, len(hashedCodes))
	for _, code := range hashedCodes {
		codes[code] = false // false = not yet used
	}
	return &RecoveryCodeSet{
		codes: codes,
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "RecoveryCodeSet"}),
	}
}

// Validate checks if a code is valid and marks it as used.
// Returns true if the code was valid and unused.
func (s *RecoveryCodeSet) Validate(code string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(code), "-", "")

	// Hash the input code to compare with stored hashes
	hash := sha256.Sum256([]byte(normalized))
	hashedInput := hex.EncodeToString(hash[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	used, exists := s.codes[hashedInput]
	if !exists || used {
		return false
	}

	s.codes[hashedInput] = true // Mark as used
	return true
}

// RemainingCount returns the number of unused codes.
func (s *RecoveryCodeSet) RemainingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, used := range s.codes {
		if !used {
			count++
		}
	}
	return count
}

// GetUsedCodes returns a list of used code hashes.
func (s *RecoveryCodeSet) GetUsedCodes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var used []string
	for code, isUsed := range s.codes {
		if isUsed {
			used = append(used, code)
		}
	}
	return used
}
