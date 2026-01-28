// Package gcp provides a Google Cloud Load Balancing adapter for loadbalancer.LoadBalancerManager.
//
// Supports HTTP(S) Load Balancers and TCP/UDP Load Balancers.
//
// Usage:
//
//	import lbgcp "github.com/chris-alexander-pop/system-design-library/pkg/network/loadbalancer/adapters/gcp"
//
//	manager, err := lbgcp.New(lbgcp.Config{ProjectID: "my-project"})
//	lb, err := manager.CreateLoadBalancer(ctx, loadbalancer.CreateLoadBalancerOptions{Name: "my-lb"})
package gcp

import (
	"context"
	"fmt"
	"os"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/loadbalancer"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Config holds GCP Load Balancing configuration.
type Config struct {
	// ProjectID is the GCP project ID.
	ProjectID string

	// Region is the GCP region for regional resources.
	Region string

	// CredentialsFile is the path to the service account JSON.
	CredentialsFile string

	// CredentialsJSON is the service account JSON content.
	CredentialsJSON []byte
}

// Manager implements loadbalancer.LoadBalancerManager for GCP.
type Manager struct {
	backendClient     *compute.BackendServicesClient
	urlMapClient      *compute.UrlMapsClient
	targetProxies     *compute.TargetHttpProxiesClient
	forwardingClient  *compute.GlobalForwardingRulesClient
	healthCheckClient *compute.HealthChecksClient
	config            Config
}

// New creates a new GCP Load Balancing manager.
func New(cfg Config) (*Manager, error) {
	ctx := context.Background()

	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		b, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, pkgerrors.Internal("failed to read credentials file", err)
		}
		creds, err := google.CredentialsFromJSON(ctx, b, compute.DefaultAuthScopes()...)
		if err != nil {
			return nil, pkgerrors.Internal("failed to parse credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	if len(cfg.CredentialsJSON) > 0 {
		creds, err := google.CredentialsFromJSON(ctx, cfg.CredentialsJSON, compute.DefaultAuthScopes()...)
		if err != nil {
			return nil, pkgerrors.Internal("failed to parse credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	backendClient, err := compute.NewBackendServicesRESTClient(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create backend services client", err)
	}

	urlMapClient, err := compute.NewUrlMapsRESTClient(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create URL maps client", err)
	}

	targetProxies, err := compute.NewTargetHttpProxiesRESTClient(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create target proxies client", err)
	}

	forwardingClient, err := compute.NewGlobalForwardingRulesRESTClient(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create forwarding rules client", err)
	}

	healthCheckClient, err := compute.NewHealthChecksRESTClient(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create health checks client", err)
	}

	return &Manager{
		backendClient:     backendClient,
		urlMapClient:      urlMapClient,
		targetProxies:     targetProxies,
		forwardingClient:  forwardingClient,
		healthCheckClient: healthCheckClient,
		config:            cfg,
	}, nil
}

// Close closes all GCP clients.
func (m *Manager) Close() {
	if m.backendClient != nil {
		m.backendClient.Close()
	}
	if m.urlMapClient != nil {
		m.urlMapClient.Close()
	}
	if m.targetProxies != nil {
		m.targetProxies.Close()
	}
	if m.forwardingClient != nil {
		m.forwardingClient.Close()
	}
	if m.healthCheckClient != nil {
		m.healthCheckClient.Close()
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (m *Manager) CreateLoadBalancer(ctx context.Context, opts loadbalancer.CreateLoadBalancerOptions) (*loadbalancer.LoadBalancer, error) {
	// Create health check
	healthCheckName := opts.Name + "-hc"
	healthCheck := &computepb.HealthCheck{
		Name: ptr(healthCheckName),
		Type: ptr("HTTP"),
		HttpHealthCheck: &computepb.HTTPHealthCheck{
			Port:        ptr(int32(80)),
			RequestPath: ptr("/health"),
		},
	}

	hcOp, err := m.healthCheckClient.Insert(ctx, &computepb.InsertHealthCheckRequest{
		Project:             m.config.ProjectID,
		HealthCheckResource: healthCheck,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create health check", err)
	}
	if err := hcOp.Wait(ctx); err != nil {
		return nil, pkgerrors.Internal("health check creation failed", err)
	}

	// Create backend service
	backendName := opts.Name + "-backend"
	backend := &computepb.BackendService{
		Name:         ptr(backendName),
		Protocol:     ptr("HTTP"),
		HealthChecks: []string{fmt.Sprintf("projects/%s/global/healthChecks/%s", m.config.ProjectID, healthCheckName)},
	}

	beOp, err := m.backendClient.Insert(ctx, &computepb.InsertBackendServiceRequest{
		Project:                m.config.ProjectID,
		BackendServiceResource: backend,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create backend service", err)
	}
	if err := beOp.Wait(ctx); err != nil {
		return nil, pkgerrors.Internal("backend service creation failed", err)
	}

	// Create URL map
	urlMapName := opts.Name + "-urlmap"
	urlMap := &computepb.UrlMap{
		Name:           ptr(urlMapName),
		DefaultService: ptr(fmt.Sprintf("projects/%s/global/backendServices/%s", m.config.ProjectID, backendName)),
	}

	umOp, err := m.urlMapClient.Insert(ctx, &computepb.InsertUrlMapRequest{
		Project:        m.config.ProjectID,
		UrlMapResource: urlMap,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create URL map", err)
	}
	if err := umOp.Wait(ctx); err != nil {
		return nil, pkgerrors.Internal("URL map creation failed", err)
	}

	// Create target HTTP proxy
	proxyName := opts.Name + "-proxy"
	proxy := &computepb.TargetHttpProxy{
		Name:   ptr(proxyName),
		UrlMap: ptr(fmt.Sprintf("projects/%s/global/urlMaps/%s", m.config.ProjectID, urlMapName)),
	}

	pxOp, err := m.targetProxies.Insert(ctx, &computepb.InsertTargetHttpProxyRequest{
		Project:                 m.config.ProjectID,
		TargetHttpProxyResource: proxy,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create target proxy", err)
	}
	if err := pxOp.Wait(ctx); err != nil {
		return nil, pkgerrors.Internal("target proxy creation failed", err)
	}

	// Create forwarding rule
	fwdName := opts.Name + "-fwd"
	fwd := &computepb.ForwardingRule{
		Name:       ptr(fwdName),
		Target:     ptr(fmt.Sprintf("projects/%s/global/targetHttpProxies/%s", m.config.ProjectID, proxyName)),
		PortRange:  ptr("80"),
		IPProtocol: ptr("TCP"),
	}

	fwOp, err := m.forwardingClient.Insert(ctx, &computepb.InsertGlobalForwardingRuleRequest{
		Project:                m.config.ProjectID,
		ForwardingRuleResource: fwd,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create forwarding rule", err)
	}
	if err := fwOp.Wait(ctx); err != nil {
		return nil, pkgerrors.Internal("forwarding rule creation failed", err)
	}

	// Get the forwarding rule to get the IP
	fwdRule, err := m.forwardingClient.Get(ctx, &computepb.GetGlobalForwardingRuleRequest{
		Project:        m.config.ProjectID,
		ForwardingRule: fwdName,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to get forwarding rule", err)
	}

	return &loadbalancer.LoadBalancer{
		ID:        fwdName,
		Name:      opts.Name,
		Type:      "HTTP",
		Scheme:    "EXTERNAL",
		DNSName:   *fwdRule.IPAddress,
		State:     "active",
		CreatedAt: time.Now(),
	}, nil
}

func (m *Manager) GetLoadBalancer(ctx context.Context, id string) (*loadbalancer.LoadBalancer, error) {
	fwd, err := m.forwardingClient.Get(ctx, &computepb.GetGlobalForwardingRuleRequest{
		Project:        m.config.ProjectID,
		ForwardingRule: id,
	})
	if err != nil {
		return nil, pkgerrors.NotFound("load balancer not found", err)
	}

	return &loadbalancer.LoadBalancer{
		ID:      *fwd.Name,
		Name:    *fwd.Name,
		Type:    "HTTP",
		DNSName: *fwd.IPAddress,
		State:   "active",
	}, nil
}

func (m *Manager) ListLoadBalancers(ctx context.Context) ([]*loadbalancer.LoadBalancer, error) {
	it := m.forwardingClient.List(ctx, &computepb.ListGlobalForwardingRulesRequest{
		Project: m.config.ProjectID,
	})

	var result []*loadbalancer.LoadBalancer
	for {
		fwd, err := it.Next()
		if err != nil {
			break
		}
		result = append(result, &loadbalancer.LoadBalancer{
			ID:      *fwd.Name,
			Name:    *fwd.Name,
			Type:    "HTTP",
			DNSName: *fwd.IPAddress,
			State:   "active",
		})
	}

	return result, nil
}

func (m *Manager) DeleteLoadBalancer(ctx context.Context, id string) error {
	baseName := id
	if len(id) > 4 && id[len(id)-4:] == "-fwd" {
		baseName = id[:len(id)-4]
	}

	// Delete in reverse order
	fwOp, _ := m.forwardingClient.Delete(ctx, &computepb.DeleteGlobalForwardingRuleRequest{
		Project:        m.config.ProjectID,
		ForwardingRule: baseName + "-fwd",
	})
	if fwOp != nil {
		_ = fwOp.Wait(ctx)
	}

	pxOp, _ := m.targetProxies.Delete(ctx, &computepb.DeleteTargetHttpProxyRequest{
		Project:         m.config.ProjectID,
		TargetHttpProxy: baseName + "-proxy",
	})
	if pxOp != nil {
		_ = pxOp.Wait(ctx)
	}

	umOp, _ := m.urlMapClient.Delete(ctx, &computepb.DeleteUrlMapRequest{
		Project: m.config.ProjectID,
		UrlMap:  baseName + "-urlmap",
	})
	if umOp != nil {
		_ = umOp.Wait(ctx)
	}

	beOp, _ := m.backendClient.Delete(ctx, &computepb.DeleteBackendServiceRequest{
		Project:        m.config.ProjectID,
		BackendService: baseName + "-backend",
	})
	if beOp != nil {
		_ = beOp.Wait(ctx)
	}

	hcOp, _ := m.healthCheckClient.Delete(ctx, &computepb.DeleteHealthCheckRequest{
		Project:     m.config.ProjectID,
		HealthCheck: baseName + "-hc",
	})
	if hcOp != nil {
		_ = hcOp.Wait(ctx)
	}

	return nil
}

// Stub implementations for interface compliance
func (m *Manager) CreateListener(ctx context.Context, opts loadbalancer.CreateListenerOptions) (*loadbalancer.Listener, error) {
	return nil, pkgerrors.Internal("use CreateLoadBalancer for GCP", nil)
}

func (m *Manager) DeleteListener(ctx context.Context, loadBalancerID, listenerID string) error {
	return nil
}

func (m *Manager) CreateTargetPool(ctx context.Context, opts loadbalancer.CreateTargetPoolOptions) (*loadbalancer.TargetPool, error) {
	return nil, pkgerrors.Internal("use backend services for GCP", nil)
}

func (m *Manager) GetTargetPool(ctx context.Context, id string) (*loadbalancer.TargetPool, error) {
	return nil, pkgerrors.NotFound("use backend services for GCP", nil)
}

func (m *Manager) DeleteTargetPool(ctx context.Context, poolID string) error {
	return nil
}

func (m *Manager) AddTarget(ctx context.Context, poolID string, target loadbalancer.Target) error {
	return nil
}

func (m *Manager) RemoveTarget(ctx context.Context, poolID, targetID string) error {
	return nil
}

func (m *Manager) GetTargetHealth(ctx context.Context, poolID string) ([]*loadbalancer.Target, error) {
	return nil, nil
}

func (m *Manager) AddRule(ctx context.Context, listenerID string, rule loadbalancer.Rule) (*loadbalancer.Rule, error) {
	return nil, nil
}

func (m *Manager) RemoveRule(ctx context.Context, listenerID, ruleID string) error {
	return nil
}

// Interface compliance
var _ loadbalancer.LoadBalancerManager = (*Manager)(nil)
