package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/otp"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// MFAProvider implements mfa.Provider using Redis.
type MFAProvider struct {
	client     *redis.Client
	totpConfig otp.TOTPConfig
}

// New creates a new Redis MFA provider.
func New(client *redis.Client, cfg mfa.Config) *MFAProvider {
	return &MFAProvider{
		client: client,
		totpConfig: otp.TOTPConfig{
			Issuer: cfg.TOTPIssuer,
			Digits: cfg.TOTPDigits,
			Period: cfg.TOTPPeriod,
		},
	}
}

func (p *MFAProvider) key(userID string) string {
	return fmt.Sprintf("auth:mfa:%s", userID)
}

func (p *MFAProvider) usedKey(userID string, counter uint64) string {
	return fmt.Sprintf("auth:mfa:used:%s:%d", userID, counter)
}

// redisEnrollment is a local struct to handle JSON serialization for Redis,
// allowing access to fields hidden in mfa.Enrollment (like Secret and Recovery).
type redisEnrollment struct {
	UserID     string                 `json:"user_id"`
	Type       string                 `json:"type"`
	Secret     string                 `json:"secret"`
	Enabled    bool                   `json:"enabled"`
	Recovery   []string               `json:"recovery"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	LastUsedAt time.Time              `json:"last_used_at,omitempty"`
}

func toRedisEnrollment(e *mfa.Enrollment) *redisEnrollment {
	return &redisEnrollment{
		UserID:     e.UserID,
		Type:       e.Type,
		Secret:     e.Secret,
		Enabled:    e.Enabled,
		Recovery:   e.Recovery,
		Metadata:   e.Metadata,
		CreatedAt:  e.CreatedAt,
		LastUsedAt: e.LastUsedAt,
	}
}

func (r *redisEnrollment) toEnrollment() *mfa.Enrollment {
	return &mfa.Enrollment{
		UserID:     r.UserID,
		Type:       r.Type,
		Secret:     r.Secret,
		Enabled:    r.Enabled,
		Recovery:   r.Recovery,
		Metadata:   r.Metadata,
		CreatedAt:  r.CreatedAt,
		LastUsedAt: r.LastUsedAt,
	}
}

func (p *MFAProvider) Enroll(ctx context.Context, userID string) (string, []string, error) {
	// 1. Generate TOTP Secret
	totp := otp.NewTOTP(p.totpConfig)
	secret, err := totp.GenerateSecret()
	if err != nil {
		return "", nil, errors.Internal("failed to generate totp secret", err)
	}

	// 2. Generate Recovery Codes
	recoveryMgr := otp.NewRecoveryCodeManager(otp.DefaultRecoveryCodeConfig())
	displayCodes, hashedCodes, err := recoveryMgr.GenerateCodes()
	if err != nil {
		return "", nil, errors.Internal("failed to generate recovery codes", err)
	}

	// 3. Store Enrollment (Enabled=false)
	enrollment := &mfa.Enrollment{
		UserID:    userID,
		Type:      "totp",
		Secret:    secret,
		Enabled:   false,
		Recovery:  hashedCodes,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(toRedisEnrollment(enrollment))
	if err != nil {
		return "", nil, errors.Internal("failed to marshal enrollment", err)
	}

	if err := p.client.Set(ctx, p.key(userID), data, 0).Err(); err != nil { // No expiration
		return "", nil, errors.Internal("failed to save enrollment to redis", err)
	}

	return secret, displayCodes, nil
}

func (p *MFAProvider) CompleteEnrollment(ctx context.Context, userID, code string) error {
	key := p.key(userID)

	// Transaction to read, validate, update
	err := p.client.Watch(ctx, func(tx *redis.Tx) error {
		data, err := tx.Get(ctx, key).Bytes()
		if err == redis.Nil {
			return errors.NotFound("mfa enrollment not found", nil)
		}
		if err != nil {
			return err
		}

		var re redisEnrollment
		if err := json.Unmarshal(data, &re); err != nil {
			return err
		}
		enrollment := re.toEnrollment()

		if enrollment.Enabled {
			return errors.Conflict("mfa already enabled", nil)
		}

		totp := otp.NewTOTP(p.totpConfig)
		valid, counter := totp.ValidateReturningCounter(enrollment.Secret, code)
		if !valid {
			return errors.InvalidArgument("invalid validation code", nil)
		}

		// Prevent replay attacks using Redis cache of used codes
		usedKey := p.usedKey(userID, counter)
		exists, err := p.client.Exists(ctx, usedKey).Result()
		if err != nil {
			return errors.Internal("failed to check used code", err)
		}
		if exists > 0 {
			return errors.Conflict("code already used", nil)
		}

		enrollment.Enabled = true
		newData, err := json.Marshal(toRedisEnrollment(enrollment))
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newData, 0)

			// Calculate expiration based on validity window
			expirationTime := time.Unix(int64(counter+uint64(p.totpConfig.Skew)+1)*int64(p.totpConfig.Period), 0)
			ttl := time.Until(expirationTime)
			if ttl < time.Second {
				ttl = time.Second
			}

			pipe.Set(ctx, usedKey, "1", ttl)
			return nil
		})
		return err
	}, key)

	if err != nil {
		if errors.Is(err, redis.TxFailedErr) {
			return errors.Conflict("mfa update conflict", err)
		}
		// Wrap if not standard logic error
		// Note: The Watch function can return our custom errors (NotFound, Conflict, InvalidArgument)
		// which are already nice. If it's a Redis error, we map it.
		// Since we can't easily distinguish mapped vs unmapped here without reflection or checks,
		// we just assume if it's not one of ours, it's internal.
		// But for now, let's just return err as is, assuming higher layers handle it or we did it inside.
		return err
	}
	return nil
}

func (p *MFAProvider) Verify(ctx context.Context, userID, code string) (bool, error) {
	data, err := p.client.Get(ctx, p.key(userID)).Bytes()
	if err == redis.Nil {
		return false, errors.NotFound("mfa enrollment not found", nil)
	}
	if err != nil {
		return false, errors.Internal("failed to get mfa enrollment", err)
	}

	var re redisEnrollment
	if err := json.Unmarshal(data, &re); err != nil {
		return false, errors.Internal("failed to unmarshal enrollment", err)
	}
	enrollment := re.toEnrollment()

	if !enrollment.Enabled {
		return false, errors.Forbidden("mfa not enabled", nil)
	}

	totp := otp.NewTOTP(p.totpConfig)
	valid, counter := totp.ValidateReturningCounter(enrollment.Secret, code)
	if !valid {
		return false, nil
	}

	// Prevent replay attacks using Redis cache of used codes
	usedKey := p.usedKey(userID, counter)
	exists, err := p.client.Exists(ctx, usedKey).Result()
	if err != nil {
		return false, errors.Internal("failed to check used code", err)
	}
	if exists > 0 {
		return false, errors.Conflict("code already used", nil)
	}

	// Calculate expiration based on validity window
	// The code becomes invalid when current_step > counter + skew.
	// We need to keep the record until then.
	expirationTime := time.Unix(int64(counter+uint64(p.totpConfig.Skew)+1)*int64(p.totpConfig.Period), 0)
	ttl := time.Until(expirationTime)
	if ttl < time.Second {
		ttl = time.Second
	}

	if err := p.client.Set(ctx, usedKey, "1", ttl).Err(); err != nil {
		return false, errors.Internal("failed to mark code as used", err)
	}

	return true, nil
}

func (p *MFAProvider) Recover(ctx context.Context, userID, code string) (bool, error) {
	key := p.key(userID)
	var success bool

	err := p.client.Watch(ctx, func(tx *redis.Tx) error {
		data, err := tx.Get(ctx, key).Bytes()
		if err == redis.Nil {
			return errors.NotFound("mfa enrollment not found", nil)
		}
		if err != nil {
			return err
		}

		var re redisEnrollment
		if err := json.Unmarshal(data, &re); err != nil {
			return err
		}
		enrollment := re.toEnrollment()

		if !enrollment.Enabled {
			return errors.Forbidden("mfa not enabled", nil)
		}

		// Check and consume recovery code
		// Logic updated to hash the input code before comparing
		normalized := strings.ReplaceAll(strings.ToLower(code), "-", "")
		hash := sha256.Sum256([]byte(normalized))
		hashedCode := hex.EncodeToString(hash[:])

		foundIndex := -1
		for i, h := range enrollment.Recovery {
			if h == hashedCode {
				foundIndex = i
				break
			}
		}

		if foundIndex == -1 {
			success = false
			return nil // No error, just invalid code
		}

		// Remove used code
		success = true
		enrollment.Recovery = append(enrollment.Recovery[:foundIndex], enrollment.Recovery[foundIndex+1:]...)

		newData, err := json.Marshal(toRedisEnrollment(enrollment))
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newData, 0)
			return nil
		})
		return err

	}, key)

	if err != nil {
		if errors.Is(err, redis.TxFailedErr) {
			return false, errors.Conflict("mfa update conflict", err)
		}
		return false, err
	}

	return success, nil
}

func (p *MFAProvider) Disable(ctx context.Context, userID string) error {
	return p.client.Del(ctx, p.key(userID)).Err()
}
