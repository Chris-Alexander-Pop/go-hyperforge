// Package server implements the gateway HTTP API.
package server

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/middleware"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/openapi"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	jwtauth "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/adapters/jwt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the gateway service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"gateway"`
	Port        string `env:"PORT" env-default:"8080"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`

	JWTSecret string `env:"JWT_SECRET" env-default:"dev-hyperforge-jwt-secret-change-me"`
	JWTIssuer string `env:"JWT_ISSUER" env-default:"go-hyperforge"`

	AuthServiceURL         string `env:"AUTH_SERVICE_URL" env-default:"http://127.0.0.1:8081"`
	UserServiceURL         string `env:"USER_SERVICE_URL" env-default:"http://127.0.0.1:8082"`
	PermissionServiceURL   string `env:"PERMISSION_SERVICE_URL" env-default:"http://127.0.0.1:8083"`
	NotificationServiceURL string `env:"NOTIFICATION_SERVICE_URL" env-default:"http://127.0.0.1:8084"`
	EmailServiceURL        string `env:"EMAIL_SERVICE_URL" env-default:"http://127.0.0.1:8085"`
	SMSServiceURL          string `env:"SMS_SERVICE_URL" env-default:"http://127.0.0.1:8086"`
	ProductServiceURL      string `env:"PRODUCT_SERVICE_URL" env-default:"http://127.0.0.1:8087"`
	CartServiceURL         string `env:"CART_SERVICE_URL" env-default:"http://127.0.0.1:8088"`
	OrderServiceURL        string `env:"ORDER_SERVICE_URL" env-default:"http://127.0.0.1:8089"`
	PaymentServiceURL      string `env:"PAYMENT_SERVICE_URL" env-default:"http://127.0.0.1:8090"`
	InventoryServiceURL    string `env:"INVENTORY_SERVICE_URL" env-default:"http://127.0.0.1:8091"`
	AppConfigServiceURL    string `env:"APPCONFIG_SERVICE_URL" env-default:"http://127.0.0.1:8092"`
	AuditServiceURL        string `env:"AUDIT_SERVICE_URL" env-default:"http://127.0.0.1:8093"`
	WorkflowServiceURL     string `env:"WORKFLOW_SERVICE_URL" env-default:"http://127.0.0.1:8094"`

	LLMGatewayServiceURL        string `env:"LLMGATEWAY_SERVICE_URL" env-default:"http://127.0.0.1:8095"`
	AgentRuntimeServiceURL      string `env:"AGENTRUNTIME_SERVICE_URL" env-default:"http://127.0.0.1:8096"`
	ToolRegistryServiceURL      string `env:"TOOLREGISTRY_SERVICE_URL" env-default:"http://127.0.0.1:8097"`
	ContextManagerServiceURL    string `env:"CONTEXTMANAGER_SERVICE_URL" env-default:"http://127.0.0.1:8098"`
	EmbeddingServiceURL         string `env:"EMBEDDING_SERVICE_URL" env-default:"http://127.0.0.1:8099"`
	VectorSearchServiceURL      string `env:"VECTORSEARCH_SERVICE_URL" env-default:"http://127.0.0.1:8100"`
	PromptEngineServiceURL      string `env:"PROMPTENGINE_SERVICE_URL" env-default:"http://127.0.0.1:8101"`
	MetricsCollectorServiceURL  string `env:"METRICSCOLLECTOR_SERVICE_URL" env-default:"http://127.0.0.1:8102"`
	LogAggregatorServiceURL     string `env:"LOGAGGREGATOR_SERVICE_URL" env-default:"http://127.0.0.1:8103"`
	TraceCollectorServiceURL    string `env:"TRACECOLLECTOR_SERVICE_URL" env-default:"http://127.0.0.1:8104"`
	AlertingServiceURL          string `env:"ALERTING_SERVICE_URL" env-default:"http://127.0.0.1:8105"`
	DiscoveryServiceURL         string `env:"DISCOVERY_SERVICE_URL" env-default:"http://127.0.0.1:8106"`
	FeatureFlagServiceURL       string `env:"FEATUREFLAG_SERVICE_URL" env-default:"http://127.0.0.1:8107"`
	SecretManagerServiceURL     string `env:"SECRETMANAGER_SERVICE_URL" env-default:"http://127.0.0.1:8108"`
	SearchServiceURL            string `env:"SEARCH_SERVICE_URL" env-default:"http://127.0.0.1:8109"`
	MediaServiceURL             string `env:"MEDIA_SERVICE_URL" env-default:"http://127.0.0.1:8110"`
	RateLimiterServiceURL       string `env:"RATELIMITER_SERVICE_URL" env-default:"http://127.0.0.1:8111"`
	PricingServiceURL           string `env:"PRICING_SERVICE_URL" env-default:"http://127.0.0.1:8112"`
	AnalyticsServiceURL         string `env:"ANALYTICS_SERVICE_URL" env-default:"http://127.0.0.1:8113"`
	ReportingServiceURL         string `env:"REPORTING_SERVICE_URL" env-default:"http://127.0.0.1:8114"`
	MLInferenceServiceURL       string `env:"MLINFERENCE_SERVICE_URL" env-default:"http://127.0.0.1:8115"`
	RecommendationServiceURL    string `env:"RECOMMENDATION_SERVICE_URL" env-default:"http://127.0.0.1:8116"`
	CMSServiceURL               string `env:"CMS_SERVICE_URL" env-default:"http://127.0.0.1:8117"`
	ScheduledJobsServiceURL     string `env:"SCHEDULEDJOBS_SERVICE_URL" env-default:"http://127.0.0.1:8118"`
	AgentOrchestratorServiceURL string `env:"AGENTORCHESTRATOR_SERVICE_URL" env-default:"http://127.0.0.1:8119"`
	FineTuningServiceURL        string `env:"FINETUNING_SERVICE_URL" env-default:"http://127.0.0.1:8120"`
	ModelRegistryServiceURL     string `env:"MODELREGISTRY_SERVICE_URL" env-default:"http://127.0.0.1:8121"`
	BillingServiceURL           string `env:"BILLING_SERVICE_URL" env-default:"http://127.0.0.1:8122"`
	InvoiceServiceURL           string `env:"INVOICE_SERVICE_URL" env-default:"http://127.0.0.1:8123"`
	TaxCalculatorServiceURL     string `env:"TAXCALCULATOR_SERVICE_URL" env-default:"http://127.0.0.1:8124"`
	SubscriptionServiceURL      string `env:"SUBSCRIPTION_SERVICE_URL" env-default:"http://127.0.0.1:8125"`
	FeedbackServiceURL          string `env:"FEEDBACK_SERVICE_URL" env-default:"http://127.0.0.1:8126"`
	IdentityProviderServiceURL  string `env:"IDENTITYPROVIDER_SERVICE_URL" env-default:"http://127.0.0.1:8127"`
	PushNotificationServiceURL  string `env:"PUSHNOTIFICATION_SERVICE_URL" env-default:"http://127.0.0.1:8128"`
	ChatServiceURL              string `env:"CHAT_SERVICE_URL" env-default:"http://127.0.0.1:8129"`
	WebhookManagerServiceURL    string `env:"WEBHOOKMANAGER_SERVICE_URL" env-default:"http://127.0.0.1:8130"`
	FraudDetectionServiceURL    string `env:"FRAUDDETECTION_SERVICE_URL" env-default:"http://127.0.0.1:8131"`
	KYCVerificationServiceURL   string `env:"KYCVERIFICATION_SERVICE_URL" env-default:"http://127.0.0.1:8132"`
	EncryptionServiceURL        string `env:"ENCRYPTION_SERVICE_URL" env-default:"http://127.0.0.1:8133"`
	KeyManagementServiceURL     string `env:"KEYMANAGEMENT_SERVICE_URL" env-default:"http://127.0.0.1:8134"`
	ComplianceServiceURL        string `env:"COMPLIANCE_SERVICE_URL" env-default:"http://127.0.0.1:8135"`
	DataRetentionServiceURL     string `env:"DATARETENTION_SERVICE_URL" env-default:"http://127.0.0.1:8136"`
	GDPRProcessorServiceURL     string `env:"GDPRPROCESSOR_SERVICE_URL" env-default:"http://127.0.0.1:8137"`
	AccessLogsServiceURL        string `env:"ACCESSLOGS_SERVICE_URL" env-default:"http://127.0.0.1:8138"`
	ETLPipelineServiceURL       string `env:"ETLPIPELINE_SERVICE_URL" env-default:"http://127.0.0.1:8139"`
	DataCatalogServiceURL       string `env:"DATACATALOG_SERVICE_URL" env-default:"http://127.0.0.1:8140"`
	SchemaRegistryServiceURL    string `env:"SCHEMAREGISTRY_SERVICE_URL" env-default:"http://127.0.0.1:8141"`
	BackupServiceURL            string `env:"BACKUP_SERVICE_URL" env-default:"http://127.0.0.1:8142"`
	ArchivalServiceURL          string `env:"ARCHIVAL_SERVICE_URL" env-default:"http://127.0.0.1:8143"`
	CachingLayerServiceURL      string `env:"CACHINGLAYER_SERVICE_URL" env-default:"http://127.0.0.1:8144"`
	BlobStorageServiceURL       string `env:"BLOBSTORAGE_SERVICE_URL" env-default:"http://127.0.0.1:8145"`
	IncidentManagerServiceURL   string `env:"INCIDENTMANAGER_SERVICE_URL" env-default:"http://127.0.0.1:8146"`
}

type route struct {
	prefix     string
	targetURL  string
	requireJWT bool
	injectUser bool
}

// Server wraps the gateway HTTP API.
type Server struct {
	rest *rest.Server
	cfg  Config
}

// New constructs the gateway HTTP server.
func New(cfg Config, tokens *jwtauth.Adapter) (*Server, error) {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg}

	verifier := auth.NewMiddlewareVerifier(tokens)
	authMW := openapi.EchoMiddleware(middleware.AuthMiddleware(verifier))

	routes := []route{
		{prefix: "/v1/auth", targetURL: cfg.AuthServiceURL, requireJWT: false},
		{prefix: "/v1/users", targetURL: cfg.UserServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/permissions", targetURL: cfg.PermissionServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/notifications", targetURL: cfg.NotificationServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/emails", targetURL: cfg.EmailServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/sms", targetURL: cfg.SMSServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/products", targetURL: cfg.ProductServiceURL, requireJWT: false},
		{prefix: "/v1/carts", targetURL: cfg.CartServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/orders", targetURL: cfg.OrderServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/payments", targetURL: cfg.PaymentServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/inventory", targetURL: cfg.InventoryServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/configs", targetURL: cfg.AppConfigServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/audits", targetURL: cfg.AuditServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/workflows", targetURL: cfg.WorkflowServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/llm-requests", targetURL: cfg.LLMGatewayServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/agents", targetURL: cfg.AgentRuntimeServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/tools", targetURL: cfg.ToolRegistryServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/contexts", targetURL: cfg.ContextManagerServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/embeddings", targetURL: cfg.EmbeddingServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/vectors", targetURL: cfg.VectorSearchServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/prompts", targetURL: cfg.PromptEngineServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/metrics", targetURL: cfg.MetricsCollectorServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/logs", targetURL: cfg.LogAggregatorServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/traces", targetURL: cfg.TraceCollectorServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/alerts", targetURL: cfg.AlertingServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/services", targetURL: cfg.DiscoveryServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/flags", targetURL: cfg.FeatureFlagServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/secrets", targetURL: cfg.SecretManagerServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/search", targetURL: cfg.SearchServiceURL, requireJWT: false},
		{prefix: "/v1/media", targetURL: cfg.MediaServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/ratelimits", targetURL: cfg.RateLimiterServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/prices", targetURL: cfg.PricingServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/analytics", targetURL: cfg.AnalyticsServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/reports", targetURL: cfg.ReportingServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/inferences", targetURL: cfg.MLInferenceServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/recommendations", targetURL: cfg.RecommendationServiceURL, requireJWT: false, injectUser: false},
		{prefix: "/v1/pages", targetURL: cfg.CMSServiceURL, requireJWT: false, injectUser: false},
		{prefix: "/v1/jobs", targetURL: cfg.ScheduledJobsServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/orchestrations", targetURL: cfg.AgentOrchestratorServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/finetunes", targetURL: cfg.FineTuningServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/models", targetURL: cfg.ModelRegistryServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/bills", targetURL: cfg.BillingServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/invoices", targetURL: cfg.InvoiceServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/taxes", targetURL: cfg.TaxCalculatorServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/subscriptions", targetURL: cfg.SubscriptionServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/feedback", targetURL: cfg.FeedbackServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/identities", targetURL: cfg.IdentityProviderServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/pushes", targetURL: cfg.PushNotificationServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/chats", targetURL: cfg.ChatServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/webhooks", targetURL: cfg.WebhookManagerServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/fraud", targetURL: cfg.FraudDetectionServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/kyc", targetURL: cfg.KYCVerificationServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/encryption", targetURL: cfg.EncryptionServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/keys", targetURL: cfg.KeyManagementServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/compliance", targetURL: cfg.ComplianceServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/retention", targetURL: cfg.DataRetentionServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/gdpr", targetURL: cfg.GDPRProcessorServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/access-logs", targetURL: cfg.AccessLogsServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/etl", targetURL: cfg.ETLPipelineServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/catalogs", targetURL: cfg.DataCatalogServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/schemas", targetURL: cfg.SchemaRegistryServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/backups", targetURL: cfg.BackupServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/archives", targetURL: cfg.ArchivalServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/caches", targetURL: cfg.CachingLayerServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/blobs", targetURL: cfg.BlobStorageServiceURL, requireJWT: true, injectUser: true},
		{prefix: "/v1/incidents", targetURL: cfg.IncidentManagerServiceURL, requireJWT: true, injectUser: true},
	}

	e := r.Echo()
	e.GET("/healthz", s.health)

	for _, rt := range routes {
		proxy, err := newProxy(rt.targetURL, rt.injectUser)
		if err != nil {
			return nil, err
		}
		handler := echo.WrapHandler(proxy)
		if rt.requireJWT {
			e.Any(rt.prefix, handler, authMW)
			e.Any(rt.prefix+"/*", handler, authMW)
		} else {
			e.Any(rt.prefix, handler)
			e.Any(rt.prefix+"/*", handler)
		}
	}

	return s, nil
}

// Echo exposes the underlying Echo instance.
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error { return s.rest.Shutdown(ctx) }

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func newProxy(rawURL string, injectUser bool) (http.Handler, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.InvalidArgument("invalid upstream URL", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	if !injectUser {
		return proxy, nil
	}
	original := proxy.Director
	proxy.Director = func(req *http.Request) {
		original(req)
		sub := middleware.GetSubject(req.Context())
		roles := middleware.GetRoles(req.Context())
		req.Header.Del("Authorization")
		if sub != "" {
			req.Header.Set("X-User-ID", sub)
		}
		if len(roles) > 0 {
			req.Header.Set("X-User-Roles", strings.Join(roles, ","))
		}
	}
	return proxy, nil
}
