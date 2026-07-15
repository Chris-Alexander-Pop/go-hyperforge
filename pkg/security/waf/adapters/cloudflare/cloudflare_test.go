package cloudflare_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/waf/adapters/cloudflare"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"github.com/google/uuid"
)

type cfBackend struct {
	mu     sync.Mutex
	token  string
	zoneID string
	rules  map[string]map[string]interface{}
}

func newCFBackend(token, zoneID string) *cfBackend {
	return &cfBackend{
		token:  token,
		zoneID: zoneID,
		rules:  make(map[string]map[string]interface{}),
	}
}

func (b *cfBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+b.token {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false})
		return
	}

	prefix := "/zones/" + b.zoneID + "/firewall/access_rules/rules"
	path := r.URL.Path

	b.mu.Lock()
	defer b.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && path == prefix:
		list := make([]map[string]interface{}, 0, len(b.rules))
		for _, rule := range b.rules {
			list = append(list, rule)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": list})
	case r.Method == http.MethodPost && path == prefix:
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		_ = json.Unmarshal(body, &payload)
		id := uuid.NewString()
		payload["id"] = id
		b.rules[id] = payload
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": payload})
	case r.Method == http.MethodDelete && strings.HasPrefix(path, prefix+"/"):
		id := strings.TrimPrefix(path, prefix+"/")
		if _, ok := b.rules[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false})
			return
		}
		delete(b.rules, id)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]string{"id": id}})
	default:
		http.NotFound(w, r)
	}
}

type CloudflareSuite struct {
	test.Suite
	srv *httptest.Server
	mgr *cloudflare.Manager
}

func (s *CloudflareSuite) SetupTest() {
	s.Suite.SetupTest()
	backend := newCFBackend("tok", "zone-1")
	s.srv = httptest.NewServer(backend)
	var err error
	s.mgr, err = cloudflare.New(cloudflare.Config{
		APIToken:   "tok",
		ZoneID:     "zone-1",
		BaseURL:    s.srv.URL,
		HTTPClient: s.srv.Client(),
	})
	s.Require().NoError(err)
}

func (s *CloudflareSuite) TearDownTest() {
	if s.srv != nil {
		s.srv.Close()
	}
}

func (s *CloudflareSuite) TestBlockAllowList() {
	s.Require().NoError(s.mgr.BlockIP(s.Ctx, "203.0.113.10", "abuse"))
	rules, err := s.mgr.GetRules(s.Ctx)
	s.Require().NoError(err)
	s.Require().Len(rules, 1)
	s.Equal("203.0.113.10", rules[0].IP)
	s.Equal("block", rules[0].Action)

	s.Require().NoError(s.mgr.AllowIP(s.Ctx, "203.0.113.10"))
	rules, err = s.mgr.GetRules(s.Ctx)
	s.Require().NoError(err)
	s.Empty(rules)
}

func (s *CloudflareSuite) TestAllowNotFound() {
	err := s.mgr.AllowIP(s.Ctx, "203.0.113.99")
	s.Require().Error(err)
	s.True(errors.Is(err, waf.ErrNotFound))
}

func (s *CloudflareSuite) TestEmptyIP() {
	err := s.mgr.BlockIP(s.Ctx, "", "x")
	s.Require().Error(err)
	s.True(errors.Is(err, waf.ErrInvalidRule))
}

func (s *CloudflareSuite) TestConfigValidate() {
	_, err := cloudflare.New(cloudflare.Config{})
	s.Require().Error(err)
}

func (s *CloudflareSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	err := s.mgr.BlockIP(ctx, "1.1.1.1", "x")
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func TestCloudflareSuite(t *testing.T) {
	test.Run(t, new(CloudflareSuite))
}
