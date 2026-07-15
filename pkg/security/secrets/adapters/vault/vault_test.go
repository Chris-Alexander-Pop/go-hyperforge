package vault_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/secrets"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/secrets/adapters/vault"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type vaultBackend struct {
	mu      sync.Mutex
	secrets map[string]string
	token   string
}

func newVaultBackend(token string) *vaultBackend {
	return &vaultBackend{secrets: make(map[string]string), token: token}
}

func (b *vaultBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Vault-Token") != b.token {
		http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1/secret/data/")
	if path == r.URL.Path || path == "" {
		http.NotFound(w, r)
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		val, ok := b.secrets[path]
		if !ok {
			http.Error(w, `{"errors":[]}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{"value": val},
			},
		})
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		var payload struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		v, _ := payload.Data["value"].(string)
		b.secrets[path] = v
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type VaultSuite struct {
	test.Suite
	srv     *httptest.Server
	backend *vaultBackend
	mgr     *vault.Manager
}

func (s *VaultSuite) SetupTest() {
	s.Suite.SetupTest()
	s.backend = newVaultBackend("test-token")
	s.srv = httptest.NewServer(s.backend)
	var err error
	s.mgr, err = vault.New(vault.Config{
		Address:    s.srv.URL,
		Token:      "test-token",
		Mount:      "secret",
		HTTPClient: s.srv.Client(),
	})
	s.Require().NoError(err)
}

func (s *VaultSuite) TearDownTest() {
	if s.srv != nil {
		s.srv.Close()
	}
}

func (s *VaultSuite) TestSetGetRotate() {
	s.Require().NoError(s.mgr.Set(s.Ctx, "db/password", "s3cr3t"))
	got, err := s.mgr.Get(s.Ctx, "db/password")
	s.Require().NoError(err)
	s.Equal("s3cr3t", got)

	rotated, err := s.mgr.Rotate(s.Ctx, "db/password", "new-secret")
	s.Require().NoError(err)
	s.Equal("new-secret", rotated)

	got, err = s.mgr.Get(s.Ctx, "db/password")
	s.Require().NoError(err)
	s.Equal("new-secret", got)
}

func (s *VaultSuite) TestRotateGenerates() {
	s.Require().NoError(s.mgr.Set(s.Ctx, "api-key", "old"))
	rotated, err := s.mgr.Rotate(s.Ctx, "api-key", "")
	s.Require().NoError(err)
	s.NotEmpty(rotated)
	s.NotEqual("old", rotated)
}

func (s *VaultSuite) TestGetNotFound() {
	_, err := s.mgr.Get(s.Ctx, "missing")
	s.Require().Error(err)
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *VaultSuite) TestRotateNotFound() {
	_, err := s.mgr.Rotate(s.Ctx, "missing", "x")
	s.Require().Error(err)
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *VaultSuite) TestEmptyName() {
	_, err := s.mgr.Get(s.Ctx, "")
	s.Require().Error(err)
	s.True(errors.Is(err, secrets.ErrInvalidArgument))
}

func (s *VaultSuite) TestBadToken() {
	mgr, err := vault.New(vault.Config{
		Address:    s.srv.URL,
		Token:      "wrong",
		HTTPClient: s.srv.Client(),
	})
	s.Require().NoError(err)
	err = mgr.Set(s.Ctx, "x", "y")
	s.Require().Error(err)
	s.True(errors.IsCode(err, secrets.CodeUnavailable))
}

func (s *VaultSuite) TestConfigValidate() {
	_, err := vault.New(vault.Config{})
	s.Require().Error(err)
}

func (s *VaultSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	_, err := s.mgr.Get(ctx, "x")
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func TestVaultSuite(t *testing.T) {
	test.Run(t, new(VaultSuite))
}
