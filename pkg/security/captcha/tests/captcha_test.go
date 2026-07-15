package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/security"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/captcha"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/captcha/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/security/captcha/adapters/recaptcha"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type CaptchaTestSuite struct {
	test.Suite
	verifier captcha.Verifier
}

func (s *CaptchaTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.verifier = memory.New("valid-token")
}

func (s *CaptchaTestSuite) TestVerify() {
	err := s.verifier.Verify(s.Ctx, "valid-token")
	s.NoError(err)
}

func (s *CaptchaTestSuite) TestVerify_Invalid() {
	err := s.verifier.Verify(s.Ctx, "bad-token")
	s.Error(err)
	s.True(errors.IsCode(err, captcha.CodeInvalidToken))
}

func (s *CaptchaTestSuite) TestConfigValidate() {
	cfg := captcha.DefaultConfig()
	s.NoError(cfg.Validate())

	bad := captcha.Config{Provider: security.ProviderRecaptcha}
	s.Error(bad.Validate())
}

func (s *CaptchaTestSuite) TestRecaptchaAdapter() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		s.Equal("test-secret", r.Form.Get("secret"))
		s.Equal("ok-token", r.Form.Get("response"))
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "score": 0.9})
	}))
	defer srv.Close()

	v, err := recaptcha.New(recaptcha.Config{
		SecretKey:     "test-secret",
		SiteVerifyURL: srv.URL,
		HTTPClient:    srv.Client(),
	})
	s.NoError(err)
	s.NoError(v.Verify(context.Background(), "ok-token"))

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false})
	}))
	defer failSrv.Close()

	v2, err := recaptcha.New(recaptcha.Config{
		SecretKey:     "test-secret",
		SiteVerifyURL: failSrv.URL,
		HTTPClient:    failSrv.Client(),
	})
	s.NoError(err)
	err = v2.Verify(context.Background(), "bad")
	s.Error(err)
	s.True(errors.IsCode(err, captcha.CodeInvalidToken))
}

func TestCaptchaSuite(t *testing.T) {
	test.Run(t, new(CaptchaTestSuite))
}
