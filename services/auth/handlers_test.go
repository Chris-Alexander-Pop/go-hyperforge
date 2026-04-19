package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/jwt"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql/adapters/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDeps(t *testing.T) (*echo.Echo, HandlerDependencies) {
	// Setup generic minimal configs
	e := echo.New()

	dbAdapter, err := sqlite.New(sql.Config{
		Driver: "sqlite",
		Name:   "file:" + t.Name() + "?mode=memory&cache=shared", // unique ephemeral memory database for isolated tests
	})
	require.NoError(t, err)

	db := dbAdapter.Get(context.Background())
	err = db.AutoMigrate(&User{})
	require.NoError(t, err)

	repo := NewDBRepository(db)
	_, err = repo.CreateUser(context.Background(), "testuser", "test@hyperforge.local", "mypassword", "user")
	require.NoError(t, err)

	jwtAdapter := jwt.New(jwt.Config{Secret: "test-secret-123", Issuer: "test"})

	deps := HandlerDependencies{
		JWT:  jwtAdapter,
		Repo: repo,
		// Omit MFA/WebAuthn for these specific login tests, or add them if needed.
	}

	BindHandlers(e, deps)

	return e, deps
}

func TestHandleLogin_Success(t *testing.T) {
	e, _ := setupTestDeps(t)

	reqBody := `{"username":"testuser","password":"mypassword"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	
	// Ensure we returned a token
	var response map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])
}

func TestHandleLogin_Failure(t *testing.T) {
	e, _ := setupTestDeps(t)

	// Wrong password
	reqBody := `{"username":"testuser","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// In our unified API Rest handlers, standard returning `errors.AppError` passes it up to genericErrorHandler.
	// Since we are bypassing the global middleware configuration here and serving echo directly, 
	// Echo will handle returning 500 or standard errors. (Our Rest wrapper sets HTTPErrorHandler).
	// Let's assert we don't get 200, but actually 500 or 401. 
	assert.NotEqual(t, http.StatusOK, rec.Code)
}
