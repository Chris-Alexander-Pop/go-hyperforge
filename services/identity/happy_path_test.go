package identity_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtauth "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/adapters/jwt"
	authserver "github.com/chris-alexander-pop/go-hyperforge/services/auth/server"
	gwserver "github.com/chris-alexander-pop/go-hyperforge/services/gateway/server"
	userserver "github.com/chris-alexander-pop/go-hyperforge/services/user/server"
)

func TestHappyPathRegisterLoginMe(t *testing.T) {
	const secret = "test-jwt-secret-for-happy-path"
	const issuer = "go-hyperforge-test"

	userSrv := userserver.New(userserver.Config{Port: "0"})
	userTS := httptest.NewServer(userSrv.Echo())
	t.Cleanup(userTS.Close)

	tokens := jwtauth.New(jwtauth.Config{
		Secret:     secret,
		Issuer:     issuer,
		Expiration: time.Hour,
	})
	authSrv := authserver.New(authserver.Config{
		Port:           "0",
		JWTSecret:      secret,
		JWTIssuer:      issuer,
		JWTExpiration:  time.Hour,
		UserServiceURL: userTS.URL,
	}, tokens)
	authTS := httptest.NewServer(authSrv.Echo())
	t.Cleanup(authTS.Close)

	gw, err := gwserver.New(gwserver.Config{
		Port:           "0",
		JWTSecret:      secret,
		JWTIssuer:      issuer,
		AuthServiceURL: authTS.URL,
		UserServiceURL: userTS.URL,
	}, tokens)
	if err != nil {
		t.Fatalf("gateway.New: %v", err)
	}
	gwTS := httptest.NewServer(gw.Echo())
	t.Cleanup(gwTS.Close)

	regBody, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "s3cret-pass",
		"name":     "Alice",
	})
	regResp, err := http.Post(gwTS.URL+"/v1/auth/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", regResp.StatusCode)
	}
	var reg struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(regResp.Body).Decode(&reg); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	if reg.UserID == "" {
		t.Fatal("expected user_id")
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "s3cret-pass",
	})
	loginResp, err := http.Post(gwTS.URL+"/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", loginResp.StatusCode)
	}
	var login struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&login); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if login.AccessToken == "" || login.TokenType != "Bearer" {
		t.Fatalf("unexpected login payload: %+v", login)
	}

	req, err := http.NewRequest(http.MethodGet, gwTS.URL+"/v1/users/me", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+login.AccessToken)
	meResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d", meResp.StatusCode)
	}
	var me struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(meResp.Body).Decode(&me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if me.ID != reg.UserID || me.Email != "alice@example.com" || me.Name != "Alice" {
		t.Fatalf("unexpected profile: %+v", me)
	}

	bad, err := http.Get(gwTS.URL + "/v1/users/me")
	if err != nil {
		t.Fatalf("unauth me: %v", err)
	}
	defer bad.Body.Close()
	if bad.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth me status = %d", bad.StatusCode)
	}
}
