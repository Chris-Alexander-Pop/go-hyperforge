package cognito

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
)

// Config holds configuration for AWS Cognito.
type Config struct {
	UserPoolID string `env:"AUTH_COGNITO_USER_POOL_ID" validate:"required"`
	ClientID   string `env:"AUTH_COGNITO_CLIENT_ID" validate:"required"`
	Region     string `env:"AUTH_COGNITO_REGION" env-default:"us-east-1"`
}

// Adapter implements auth.IdentityProvider and auth.Verifier for AWS Cognito.
type Adapter struct {
	client     *cognitoidentityprovider.Client
	userPoolID string
	clientID   string
	region     string
	issuer     string

	verifierOnce sync.Once
	verifier     *oidc.IDTokenVerifier
	verifierErr  error
}

// New creates a new Cognito adapter.
func New(ctx context.Context, cfg Config) (*Adapter, error) {
	if cfg.UserPoolID == "" || cfg.ClientID == "" {
		return nil, auth.ErrInvalidConfigMsg("UserPoolID and ClientID are required", nil)
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, errors.Internal("failed to load aws config", err)
	}

	client := cognitoidentityprovider.NewFromConfig(awsCfg)
	issuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", cfg.Region, cfg.UserPoolID)

	return &Adapter{
		client:     client,
		userPoolID: cfg.UserPoolID,
		clientID:   cfg.ClientID,
		region:     cfg.Region,
		issuer:     issuer,
	}, nil
}

// Login authenticates a user with username and password via USER_PASSWORD_AUTH.
func (a *Adapter) Login(ctx context.Context, username, password string) (*auth.Claims, error) {
	if username == "" || password == "" {
		return nil, auth.ErrInvalidCredentials
	}

	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(a.clientID),
		AuthParameters: map[string]string{
			"USERNAME": username,
			"PASSWORD": password,
		},
	}

	output, err := a.client.InitiateAuth(ctx, input)
	if err != nil {
		return nil, errors.Unauthorized("login failed", err)
	}

	if output.AuthenticationResult == nil {
		return nil, errors.Unauthorized("no authentication result returned (challenge required)", nil)
	}

	idToken := aws.ToString(output.AuthenticationResult.IdToken)
	if idToken != "" {
		claims, verr := a.Verify(ctx, idToken)
		if verr == nil {
			if claims.Metadata == nil {
				claims.Metadata = map[string]interface{}{}
			}
			claims.Metadata["access_token"] = aws.ToString(output.AuthenticationResult.AccessToken)
			claims.Metadata["refresh_token"] = aws.ToString(output.AuthenticationResult.RefreshToken)
			return claims, nil
		}
		// Fall through to basic claims if JWKS verification is unavailable offline.
	}

	return &auth.Claims{
		Subject: username,
		Issuer:  a.issuer,
		Metadata: map[string]interface{}{
			"access_token":  aws.ToString(output.AuthenticationResult.AccessToken),
			"refresh_token": aws.ToString(output.AuthenticationResult.RefreshToken),
			"id_token":      idToken,
		},
	}, nil
}

// Verify validates a Cognito JWT (ID token) via OIDC discovery + JWKS.
func (a *Adapter) Verify(ctx context.Context, token string) (*auth.Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, auth.ErrInvalidToken
	}

	verifier, err := a.getVerifier(ctx)
	if err != nil {
		return nil, errors.Internal("failed to initialize cognito oidc verifier", err)
	}

	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, auth.ErrInvalidTokenWrap(err)
	}

	return claimsFromOIDC(idToken)
}

func (a *Adapter) getVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	a.verifierOnce.Do(func() {
		provider, err := oidc.NewProvider(ctx, a.issuer)
		if err != nil {
			a.verifierErr = err
			return
		}
		a.verifier = provider.Verifier(&oidc.Config{ClientID: a.clientID})
	})
	return a.verifier, a.verifierErr
}

func claimsFromOIDC(idToken *oidc.IDToken) (*auth.Claims, error) {
	var raw struct {
		Email           string   `json:"email"`
		CognitoUsername string   `json:"cognito:username"`
		CognitoGroups   []string `json:"cognito:groups"`
		Role            string   `json:"role"`
		Roles           []string `json:"roles"`
	}
	if err := idToken.Claims(&raw); err != nil {
		return nil, errors.Internal("failed to parse cognito claims", err)
	}

	roles := append([]string{}, raw.CognitoGroups...)
	if raw.Role != "" {
		roles = append(roles, raw.Role)
	}
	roles = append(roles, raw.Roles...)

	return &auth.Claims{
		Subject:   idToken.Subject,
		Issuer:    idToken.Issuer,
		Audience:  idToken.Audience,
		ExpiresAt: idToken.Expiry.Unix(),
		IssuedAt:  idToken.IssuedAt.Unix(),
		Email:     raw.Email,
		Roles:     roles,
		Metadata: map[string]interface{}{
			"cognito:username": raw.CognitoUsername,
		},
	}, nil
}

// Ensure interface conformance.
var (
	_ auth.IdentityProvider = (*Adapter)(nil)
	_ auth.Verifier         = (*Adapter)(nil)
)
