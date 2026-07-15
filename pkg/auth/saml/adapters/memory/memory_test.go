package memory_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/saml"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/saml/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type SAMLSuite struct {
	test.Suite
	client *memory.Client
}

func (s *SAMLSuite) SetupTest() {
	s.Suite.SetupTest()
	c, err := memory.New(saml.Config{
		EntityID:    "https://sp.example.com/metadata",
		ACSURL:      "https://sp.example.com/acs",
		IdPSSOURL:   "https://idp.example.com/sso",
		IdPEntityID: "https://idp.example.com/metadata",
	})
	s.Require().NoError(err)
	s.client = c
}

func (s *SAMLSuite) TestNewRequiresConfig() {
	_, err := memory.New(saml.Config{})
	s.Error(err)
	s.True(errors.IsCode(err, errors.CodeInvalidArgument))
}

func (s *SAMLSuite) TestMetadataAndAuthnURL() {
	meta, err := s.client.MetadataXML(s.Ctx)
	s.NoError(err)
	s.Contains(string(meta), "https://sp.example.com/metadata")
	s.Contains(string(meta), "https://sp.example.com/acs")

	u, err := s.client.AuthnRequestURL(s.Ctx, "relay-1")
	s.NoError(err)
	s.True(strings.HasPrefix(u, "https://idp.example.com/sso"))
	s.Contains(u, "SAMLRequest=")
	s.Contains(u, "RelayState=relay-1")
}

func (s *SAMLSuite) TestParseResponseJSONStub() {
	payload, _ := json.Marshal(map[string]any{
		"sub":   "user-42",
		"email": "u@example.com",
		"roles": []string{"admin"},
	})
	b64 := base64.StdEncoding.EncodeToString(payload)

	claims, err := s.client.ParseResponse(s.Ctx, saml.AssertionConsumerRequest{
		SAMLResponse: b64,
		RelayState:   "ignored",
	})
	s.NoError(err)
	s.Equal("user-42", claims.Subject)
	s.Equal("u@example.com", claims.Email)
	s.Equal([]string{"admin"}, claims.Roles)
	s.Equal("https://idp.example.com/metadata", claims.Issuer)
	s.Equal([]string{"https://sp.example.com/metadata"}, claims.Audience)
}

func (s *SAMLSuite) TestParseResponseRelayState() {
	s.client.RegisterRelayState("rs", &auth.Claims{Subject: "from-relay", Email: "r@e.com"})
	claims, err := s.client.ParseResponse(s.Ctx, saml.AssertionConsumerRequest{RelayState: "rs"})
	s.NoError(err)
	s.Equal("from-relay", claims.Subject)
}

func (s *SAMLSuite) TestParseResponseInvalid() {
	_, err := s.client.ParseResponse(s.Ctx, saml.AssertionConsumerRequest{SAMLResponse: "!!!"})
	s.Error(err)
	s.True(errors.Is(err, saml.ErrInvalidResponse) || errors.IsCode(err, errors.CodeInvalidArgument))
}

func (s *SAMLSuite) TestValidateXMLSignatureUnimplemented() {
	err := s.client.ValidateXMLSignature(s.Ctx, []byte("<Response/>"))
	s.Error(err)
	s.True(errors.Is(err, saml.ErrUnimplementedSSO) || errors.IsCode(err, errors.CodeUnimplemented))
}

func (s *SAMLSuite) TestContextCancel() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	_, err := s.client.MetadataXML(ctx)
	s.Error(err)
}

func TestSAMLSuite(t *testing.T) {
	test.Run(t, new(SAMLSuite))
}
