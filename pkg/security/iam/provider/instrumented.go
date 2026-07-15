package provider

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/iam"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedIdentityProvider wraps an IdentityProvider with logging and tracing.
type InstrumentedIdentityProvider struct {
	next   IdentityProvider
	tracer trace.Tracer
}

// NewInstrumentedIdentityProvider creates a new instrumented identity provider.
func NewInstrumentedIdentityProvider(next IdentityProvider) *InstrumentedIdentityProvider {
	return &InstrumentedIdentityProvider{
		next:   next,
		tracer: otel.Tracer("pkg/iam/provider"),
	}
}

func (i *InstrumentedIdentityProvider) Authenticate(ctx context.Context, creds iam.Credentials) (*iam.User, error) {
	ctx, span := i.tracer.Start(ctx, "provider.Authenticate", trace.WithAttributes(
		attribute.String("username", creds.Username),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "authenticating user", "username", creds.Username)

	user, err := i.next.Authenticate(ctx, creds)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "authentication failed", "username", creds.Username, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("user.id", user.ID))
	logger.L().InfoContext(ctx, "user authenticated", "user_id", user.ID)
	return user, nil
}

func (i *InstrumentedIdentityProvider) IssueToken(ctx context.Context, user *iam.User, scopes []string) (*iam.Token, error) {
	ctx, span := i.tracer.Start(ctx, "provider.IssueToken", trace.WithAttributes(
		attribute.String("user.id", user.ID),
		attribute.StringSlice("scopes", scopes),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "issuing token", "user_id", user.ID)

	token, err := i.next.IssueToken(ctx, user, scopes)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to issue token", "user_id", user.ID, "error", err)
		return nil, err
	}

	logger.L().InfoContext(ctx, "token issued", "user_id", user.ID)
	return token, nil
}

func (i *InstrumentedIdentityProvider) ValidateToken(ctx context.Context, token string) (*iam.User, error) {
	ctx, span := i.tracer.Start(ctx, "provider.ValidateToken")
	defer span.End()

	// Don't log full tokens for security
	logger.L().DebugContext(ctx, "validating token")

	user, err := i.next.ValidateToken(ctx, token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("user.id", user.ID))
	return user, nil
}

func (i *InstrumentedIdentityProvider) RevokeToken(ctx context.Context, token string) error {
	ctx, span := i.tracer.Start(ctx, "provider.RevokeToken")
	defer span.End()

	logger.L().InfoContext(ctx, "revoking token")

	err := i.next.RevokeToken(ctx, token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (i *InstrumentedIdentityProvider) CreateUser(ctx context.Context, user iam.User, password string) (string, error) {
	ctx, span := i.tracer.Start(ctx, "provider.CreateUser", trace.WithAttributes(
		attribute.String("username", user.Username),
		attribute.String("email", user.Email),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating user", "username", user.Username)

	id, err := i.next.CreateUser(ctx, user, password)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create user", "username", user.Username, "error", err)
		return "", err
	}

	span.SetAttributes(attribute.String("user.id", id))
	logger.L().InfoContext(ctx, "user created", "id", id)
	return id, nil
}
