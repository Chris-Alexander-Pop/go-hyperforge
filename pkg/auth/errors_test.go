package auth_test

import (
	"encoding/base64"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestNewAESEncryptorFromKey(t *testing.T) {
	enc, err := auth.NewAESEncryptorFromKey("")
	if err != nil || enc != nil {
		t.Fatalf("empty key should disable encryption: enc=%v err=%v", enc, err)
	}

	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	enc, err = auth.NewAESEncryptorFromKey(base64.StdEncoding.EncodeToString(raw))
	if err != nil || enc == nil {
		t.Fatalf("base64 key: %v %v", enc, err)
	}
	ct, err := enc.EncryptString("hello")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := enc.DecryptString(ct)
	if err != nil || pt != "hello" {
		t.Fatalf("roundtrip: %q %v", pt, err)
	}

	enc, err = auth.NewAESEncryptorFromKey("passphrase-not-exact-length")
	if err != nil || enc == nil {
		t.Fatalf("passphrase derive: %v %v", enc, err)
	}
}

func TestErrorSentinels(t *testing.T) {
	if !auth.IsInvalidToken(auth.ErrInvalidToken) {
		t.Fatal("IsInvalidToken")
	}
	if !auth.IsInvalidCredentials(auth.ErrInvalidCredentials) {
		t.Fatal("IsInvalidCredentials")
	}
	if !auth.IsInvalidConfig(auth.ErrInvalidConfig) {
		t.Fatal("IsInvalidConfig")
	}
	wrapped := auth.ErrInvalidTokenWrap(errors.New("x", "y", nil))
	if !auth.IsInvalidToken(wrapped) {
		t.Fatal("wrapped invalid token")
	}
	cfg := auth.ErrInvalidConfigMsg("missing issuer", nil)
	if !auth.IsInvalidConfig(cfg) {
		t.Fatal("config msg")
	}
}
