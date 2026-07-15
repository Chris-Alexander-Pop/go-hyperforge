package identity_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/identity"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSIWEVerifier_VerifyValidSignature(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	privHex := fmt.Sprintf("%x", crypto.FromECDSA(key))

	v := identity.NewSIWEVerifier()
	msg, err := v.CreateMessage("example.com", addr, "https://example.com", "Sign in", 1)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := identity.SignPersonal(identity.FormatSIWE(msg), privHex)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := v.Verify(context.Background(), msg, sig)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected valid signature")
	}
}

func TestSIWEVerifier_RejectWrongAddress(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	other, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	privHex := fmt.Sprintf("%x", crypto.FromECDSA(other))

	v := identity.NewSIWEVerifier()
	msg, err := v.CreateMessage("example.com", addr, "https://example.com", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := identity.SignPersonal(identity.FormatSIWE(msg), privHex)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := v.Verify(context.Background(), msg, sig)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected signature mismatch")
	}
}

func TestSIWEVerifier_NonceReuse(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	privHex := fmt.Sprintf("%x", crypto.FromECDSA(key))

	v := identity.NewSIWEVerifier()
	msg, err := v.CreateMessage("example.com", addr, "https://example.com", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := identity.SignPersonal(identity.FormatSIWE(msg), privHex)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := v.Verify(context.Background(), msg, sig)
	if err != nil || !ok {
		t.Fatalf("first verify: ok=%v err=%v", ok, err)
	}

	_, err = v.Verify(context.Background(), msg, sig)
	if err == nil {
		t.Fatal("expected nonce reuse error")
	}
	if !pkgerrors.IsCode(err, web3.CodeNonceReused) {
		t.Fatalf("code = %s", pkgerrors.Code(err))
	}
}

func TestSIWEVerifier_ExpiredAndNotBefore(t *testing.T) {
	v := identity.NewSIWEVerifier()
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	expired := &web3.SIWEMessage{
		Domain: "example.com", Address: "0xabc", URI: "https://example.com",
		Version: "1", ChainID: 1, Nonce: "n1", IssuedAt: past, ExpirationTime: &past,
	}
	_, err := v.Verify(context.Background(), expired, "0x00")
	if !pkgerrors.IsCode(err, web3.CodeMessageExpired) {
		t.Fatalf("expired code = %s err=%v", pkgerrors.Code(err), err)
	}

	early := &web3.SIWEMessage{
		Domain: "example.com", Address: "0xabc", URI: "https://example.com",
		Version: "1", ChainID: 1, Nonce: "n2", IssuedAt: time.Now(), NotBefore: &future,
	}
	_, err = v.Verify(context.Background(), early, "0x00")
	if !pkgerrors.IsCode(err, web3.CodeMessageNotYet) {
		t.Fatalf("not-before code = %s err=%v", pkgerrors.Code(err), err)
	}
}

func TestSIWEVerifier_InvalidSignatureFormat(t *testing.T) {
	v := identity.NewSIWEVerifier()
	msg := &web3.SIWEMessage{
		Domain: "example.com", Address: "0xabc", URI: "https://example.com",
		Version: "1", ChainID: 1, Nonce: "n3", IssuedAt: time.Now(),
	}
	_, err := v.Verify(context.Background(), msg, "not-hex")
	if !pkgerrors.IsCode(err, web3.CodeInvalidSignature) {
		t.Fatalf("code = %s", pkgerrors.Code(err))
	}
}

func TestSIWEVerifier_ConcurrentNonceConsumption(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	privHex := fmt.Sprintf("%x", crypto.FromECDSA(key))

	v := identity.NewSIWEVerifier()
	msg, err := v.CreateMessage("example.com", addr, "https://example.com", "race", 1)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := identity.SignPersonal(identity.FormatSIWE(msg), privHex)
	if err != nil {
		t.Fatal(err)
	}

	const n = 32
	var wg sync.WaitGroup
	results := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := v.Verify(context.Background(), msg, sig)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var successes, reuse int
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if pkgerrors.IsCode(err, web3.CodeNonceReused) {
			reuse++
			continue
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want 1", successes)
	}
	if reuse != n-1 {
		t.Fatalf("reuse = %d, want %d", reuse, n-1)
	}
}

func TestSIWEVerifier_CanceledContext(t *testing.T) {
	v := identity.NewSIWEVerifier()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := v.Verify(ctx, &web3.SIWEMessage{Nonce: "x"}, "0x00")
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestSIWEVerifier_ImplementsWeb3Verifier(t *testing.T) {
	var _ web3.Verifier = identity.NewSIWEVerifier()
}

func TestFormatSIWE_OptionalFields(t *testing.T) {
	exp := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	nb := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	msg := &web3.SIWEMessage{
		Domain: "app.io", Address: "0xABC", Statement: "Hello",
		URI: "https://app.io", Version: "1", ChainID: 5, Nonce: "abc123",
		IssuedAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationTime: &exp, NotBefore: &nb, RequestID: "req-1",
		Resources: []string{"ipfs://x", "https://y"},
	}
	s := identity.FormatSIWE(msg)
	for _, want := range []string{
		"app.io wants you to sign in",
		"0xABC",
		"Hello",
		"URI: https://app.io",
		"Chain ID: 5",
		"Nonce: abc123",
		"Expiration Time:",
		"Not Before:",
		"Request ID: req-1",
		"- ipfs://x",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
	if identity.FormatSIWE(nil) != "" {
		t.Fatal("nil message should format empty")
	}
}

func TestVerifySignature_RoundTrip(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	privHex := fmt.Sprintf("%x", crypto.FromECDSA(key))
	msg := "hello web3"
	sig, err := identity.SignPersonal(msg, privHex)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := identity.VerifySignature(msg, sig, addr)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	ok, err = identity.VerifySignature(msg, sig, "0x0000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected address mismatch")
	}
}

func TestParseDID_AndEthereumDID(t *testing.T) {
	d, err := identity.ParseDID("did:ethr:0xAbC/path?q=1#frag")
	if err != nil {
		t.Fatal(err)
	}
	if d.Method != "ethr" || d.Identifier != "0xAbC" {
		t.Fatalf("parsed = %+v", d)
	}
	if d.Path != "path" || d.Query != "q=1" || d.Fragment != "frag" {
		t.Fatalf("optional = %+v", d)
	}
	if got := d.String(); got != "did:ethr:0xAbC/path?q=1#frag" {
		t.Fatalf("String = %q", got)
	}

	_, err = identity.ParseDID("not-a-did")
	if err == nil {
		t.Fatal("expected invalid DID")
	}

	ed := identity.EthereumDID("0xABCDEF")
	if ed.Method != "ethr" || ed.Identifier != "0xabcdef" {
		t.Fatalf("ethr DID = %+v", ed)
	}
}

func TestGenerateNonce_Unique(t *testing.T) {
	v := identity.NewSIWEVerifier()
	a, err := v.GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	b, err := v.GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || len(a) != 32 {
		t.Fatalf("a=%q b=%q", a, b)
	}
}

func TestInstrumentedVerifier(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	privHex := fmt.Sprintf("%x", crypto.FromECDSA(key))

	raw := identity.NewSIWEVerifier()
	v := web3.NewInstrumentedVerifier(raw)
	msg, err := v.CreateMessage("example.com", addr, "https://example.com", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := v.GenerateNonce()
	if err != nil || nonce == "" {
		t.Fatalf("nonce=%q err=%v", nonce, err)
	}
	sig, err := identity.SignPersonal(identity.FormatSIWE(msg), privHex)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := v.Verify(context.Background(), msg, sig)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
