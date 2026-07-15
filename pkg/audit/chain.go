package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// genesisPrevHash is the prev_hash of the first event in a chain.
const genesisPrevHash = "GENESIS"

// HashEvent computes a deterministic SHA-256 hex digest of the event for
// tamper-evident chaining. Hash and ID are excluded from the digest input;
// PrevHash is included so the chain links to the previous event.
func HashEvent(e Event) (string, error) {
	payload := struct {
		PrevHash       string                 `json:"prev_hash"`
		Timestamp      string                 `json:"timestamp"`
		EventType      EventType              `json:"event_type"`
		Outcome        Outcome                `json:"outcome"`
		ActorID        string                 `json:"actor_id,omitempty"`
		ActorType      string                 `json:"actor_type,omitempty"`
		ActorIP        string                 `json:"actor_ip,omitempty"`
		ActorUserAgent string                 `json:"actor_user_agent,omitempty"`
		TargetID       string                 `json:"target_id,omitempty"`
		TargetType     string                 `json:"target_type,omitempty"`
		ResourceID     string                 `json:"resource_id,omitempty"`
		ResourceType   string                 `json:"resource_type,omitempty"`
		Action         string                 `json:"action,omitempty"`
		Description    string                 `json:"description,omitempty"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
		RequestID      string                 `json:"request_id,omitempty"`
		SessionID      string                 `json:"session_id,omitempty"`
		CorrelationID  string                 `json:"correlation_id,omitempty"`
		ErrorCode      string                 `json:"error_code,omitempty"`
		ErrorMessage   string                 `json:"error_message,omitempty"`
	}{
		PrevHash:       e.PrevHash,
		Timestamp:      e.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		EventType:      e.EventType,
		Outcome:        e.Outcome,
		ActorID:        e.ActorID,
		ActorType:      e.ActorType,
		ActorIP:        e.ActorIP,
		ActorUserAgent: e.ActorUserAgent,
		TargetID:       e.TargetID,
		TargetType:     e.TargetType,
		ResourceID:     e.ResourceID,
		ResourceType:   e.ResourceType,
		Action:         e.Action,
		Description:    e.Description,
		Metadata:       canonicalizeMetadata(e.Metadata),
		RequestID:      e.RequestID,
		SessionID:      e.SessionID,
		CorrelationID:  e.CorrelationID,
		ErrorCode:      e.ErrorCode,
		ErrorMessage:   e.ErrorMessage,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", ErrMarshalFailed(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func canonicalizeMetadata(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]interface{}, len(m))
	for _, k := range keys {
		out[k] = m[k]
	}
	return out
}

// VerifyChain checks that events form a contiguous hash chain in order.
// Empty slices are valid. Returns ErrChainBroken on the first mismatch.
func VerifyChain(events []Event) error {
	prev := genesisPrevHash
	for i, e := range events {
		if e.PrevHash != prev {
			return ErrChainBroken(i, "prev_hash mismatch")
		}
		want, err := HashEvent(e)
		if err != nil {
			return err
		}
		if e.Hash != want {
			return ErrChainBroken(i, "hash mismatch")
		}
		prev = e.Hash
	}
	return nil
}
