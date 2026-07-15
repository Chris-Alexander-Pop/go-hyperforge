package oauth2

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedAuthorizationServer wraps AuthorizationServer with tracing/logging.
type InstrumentedAuthorizationServer struct {
	next   AuthorizationServer
	tracer trace.Tracer
}

// NewInstrumentedAuthorizationServer creates an instrumented wrapper.
func NewInstrumentedAuthorizationServer(next AuthorizationServer) *InstrumentedAuthorizationServer {
	return &InstrumentedAuthorizationServer{
		next:   next,
		tracer: otel.Tracer("pkg/auth/oauth2"),
	}
}

func (s *InstrumentedAuthorizationServer) RegisterClient(ctx context.Context, client Client) error {
	ctx, span := s.tracer.Start(ctx, "oauth2.RegisterClient", trace.WithAttributes(
		attribute.String("oauth2.client_id", client.ID),
	))
	defer span.End()
	err := s.next.RegisterClient(ctx, client)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (s *InstrumentedAuthorizationServer) Authorize(ctx context.Context, req AuthorizeRequest) (*AuthorizeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "oauth2.Authorize", trace.WithAttributes(
		attribute.String("oauth2.client_id", req.ClientID),
		attribute.String("oauth2.subject", req.Subject),
	))
	defer span.End()
	logger.L().DebugContext(ctx, "oauth2 authorize", "client_id", req.ClientID)
	resp, err := s.next.Authorize(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return resp, nil
}

func (s *InstrumentedAuthorizationServer) Token(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	ctx, span := s.tracer.Start(ctx, "oauth2.Token", trace.WithAttributes(
		attribute.String("oauth2.client_id", req.ClientID),
		attribute.String("oauth2.grant_type", string(req.GrantType)),
	))
	defer span.End()
	resp, err := s.next.Token(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().WarnContext(ctx, "oauth2 token failed", "error", err)
		return nil, err
	}
	return resp, nil
}

func (s *InstrumentedAuthorizationServer) Issuer() TokenIssuer {
	return s.next.Issuer()
}
