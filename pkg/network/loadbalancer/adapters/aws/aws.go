// Package aws provides an AWS Elastic Load Balancing adapter for loadbalancer.LoadBalancerManager.
//
// Supports both Application Load Balancers (ALB) and Network Load Balancers (NLB).
//
// Usage:
//
//	import lbaws "github.com/chris-alexander-pop/go-hyperforge/pkg/network/loadbalancer/adapters/aws"
//
//	manager, err := lbaws.New(lbaws.Config{Region: "us-east-1"})
//	lb, err := manager.CreateLoadBalancer(ctx, loadbalancer.CreateLoadBalancerOptions{Name: "my-alb"})
package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/loadbalancer"
)

// Config holds AWS ELB configuration.
type Config struct {
	// Region is the AWS region.
	Region string

	// AccessKeyID is the AWS access key.
	AccessKeyID string

	// SecretAccessKey is the AWS secret key.
	SecretAccessKey string

	// Endpoint is an optional custom endpoint (for LocalStack).
	Endpoint string
}

// Manager implements loadbalancer.LoadBalancerManager for AWS ELB.
type Manager struct {
	client *elasticloadbalancingv2.Client
	config Config
}

// New creates a new AWS ELB manager.
func New(cfg Config) (*Manager, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to load AWS config", err)
	}

	clientOpts := []func(*elasticloadbalancingv2.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *elasticloadbalancingv2.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	return &Manager{
		client: elasticloadbalancingv2.NewFromConfig(awsCfg, clientOpts...),
		config: cfg,
	}, nil
}

func (m *Manager) CreateLoadBalancer(ctx context.Context, opts loadbalancer.CreateLoadBalancerOptions) (*loadbalancer.LoadBalancer, error) {
	lbType := types.LoadBalancerTypeEnumApplication
	if opts.Type == "network" || opts.Type == "nlb" {
		lbType = types.LoadBalancerTypeEnumNetwork
	}

	scheme := types.LoadBalancerSchemeEnumInternetFacing
	if opts.Scheme == "internal" {
		scheme = types.LoadBalancerSchemeEnumInternal
	}

	input := &elasticloadbalancingv2.CreateLoadBalancerInput{
		Name:           aws.String(opts.Name),
		Type:           lbType,
		Scheme:         scheme,
		Subnets:        opts.Subnets,
		SecurityGroups: opts.Security,
	}

	if len(opts.Tags) > 0 {
		input.Tags = make([]types.Tag, 0, len(opts.Tags))
		for k, v := range opts.Tags {
			input.Tags = append(input.Tags, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
	}

	output, err := m.client.CreateLoadBalancer(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create load balancer", err)
	}

	if len(output.LoadBalancers) == 0 {
		return nil, pkgerrors.Internal("no load balancer returned", nil)
	}

	return mapAWSLoadBalancer(&output.LoadBalancers[0]), nil
}

func mapAWSLoadBalancer(lb *types.LoadBalancer) *loadbalancer.LoadBalancer {
	return &loadbalancer.LoadBalancer{
		ID:        *lb.LoadBalancerArn,
		Name:      *lb.LoadBalancerName,
		Type:      string(lb.Type),
		Scheme:    string(lb.Scheme),
		DNSName:   *lb.DNSName,
		State:     string(lb.State.Code),
		CreatedAt: *lb.CreatedTime,
	}
}

func (m *Manager) GetLoadBalancer(ctx context.Context, id string) (*loadbalancer.LoadBalancer, error) {
	output, err := m.client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{id},
	})
	if err != nil {
		return nil, pkgerrors.NotFound("load balancer not found", err)
	}

	if len(output.LoadBalancers) == 0 {
		return nil, pkgerrors.NotFound("load balancer not found", nil)
	}

	return mapAWSLoadBalancer(&output.LoadBalancers[0]), nil
}

func (m *Manager) ListLoadBalancers(ctx context.Context) ([]*loadbalancer.LoadBalancer, error) {
	output, err := m.client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list load balancers", err)
	}

	result := make([]*loadbalancer.LoadBalancer, len(output.LoadBalancers))
	for i, lb := range output.LoadBalancers {
		result[i] = mapAWSLoadBalancer(&lb)
	}

	return result, nil
}

func (m *Manager) DeleteLoadBalancer(ctx context.Context, id string) error {
	_, err := m.client.DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(id),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete load balancer", err)
	}
	return nil
}

func (m *Manager) CreateListener(ctx context.Context, opts loadbalancer.CreateListenerOptions) (*loadbalancer.Listener, error) {
	protocol := types.ProtocolEnumHttp
	switch opts.Protocol {
	case loadbalancer.ProtocolHTTPS:
		protocol = types.ProtocolEnumHttps
	case loadbalancer.ProtocolTCP:
		protocol = types.ProtocolEnumTcp
	case loadbalancer.ProtocolTLS:
		protocol = types.ProtocolEnumTls
	}

	input := &elasticloadbalancingv2.CreateListenerInput{
		LoadBalancerArn: aws.String(opts.LoadBalancerID),
		Port:            aws.Int32(int32(opts.Port)),
		Protocol:        protocol,
		DefaultActions: []types.Action{
			{
				Type:           types.ActionTypeEnumForward,
				TargetGroupArn: aws.String(opts.TargetPoolID),
			},
		},
	}

	if opts.SSLCertificateARN != "" {
		input.Certificates = []types.Certificate{
			{CertificateArn: aws.String(opts.SSLCertificateARN)},
		}
	}

	output, err := m.client.CreateListener(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create listener", err)
	}

	if len(output.Listeners) == 0 {
		return nil, pkgerrors.Internal("no listener returned", nil)
	}

	l := output.Listeners[0]
	return &loadbalancer.Listener{
		ID:             *l.ListenerArn,
		Port:           int(*l.Port),
		Protocol:       loadbalancer.Protocol(l.Protocol),
		LoadBalancerID: opts.LoadBalancerID,
		CreatedAt:      time.Now(),
	}, nil
}

func (m *Manager) DeleteListener(ctx context.Context, loadBalancerID, listenerID string) error {
	_, err := m.client.DeleteListener(ctx, &elasticloadbalancingv2.DeleteListenerInput{
		ListenerArn: aws.String(listenerID),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete listener", err)
	}
	return nil
}

func (m *Manager) CreateTargetPool(ctx context.Context, opts loadbalancer.CreateTargetPoolOptions) (*loadbalancer.TargetPool, error) {
	protocol := types.ProtocolEnumHttp
	switch opts.Protocol {
	case loadbalancer.ProtocolHTTPS:
		protocol = types.ProtocolEnumHttps
	case loadbalancer.ProtocolTCP:
		protocol = types.ProtocolEnumTcp
	}

	input := &elasticloadbalancingv2.CreateTargetGroupInput{
		Name:       aws.String(opts.Name),
		Protocol:   protocol,
		Port:       aws.Int32(int32(opts.Port)),
		TargetType: types.TargetTypeEnumInstance,
	}

	if opts.HealthCheck != nil {
		input.HealthCheckEnabled = aws.Bool(true)
		input.HealthCheckPath = aws.String(opts.HealthCheck.Path)
		input.HealthCheckIntervalSeconds = aws.Int32(int32(opts.HealthCheck.IntervalSeconds))
		input.HealthyThresholdCount = aws.Int32(int32(opts.HealthCheck.HealthyThreshold))
		input.UnhealthyThresholdCount = aws.Int32(int32(opts.HealthCheck.UnhealthyThreshold))
	}

	output, err := m.client.CreateTargetGroup(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create target group", err)
	}

	if len(output.TargetGroups) == 0 {
		return nil, pkgerrors.Internal("no target group returned", nil)
	}

	tg := output.TargetGroups[0]
	return &loadbalancer.TargetPool{
		ID:        *tg.TargetGroupArn,
		Name:      *tg.TargetGroupName,
		Protocol:  loadbalancer.Protocol(tg.Protocol),
		Port:      int(*tg.Port),
		CreatedAt: time.Now(),
	}, nil
}

func (m *Manager) GetTargetPool(ctx context.Context, id string) (*loadbalancer.TargetPool, error) {
	output, err := m.client.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{
		TargetGroupArns: []string{id},
	})
	if err != nil {
		return nil, pkgerrors.NotFound("target pool not found", err)
	}

	if len(output.TargetGroups) == 0 {
		return nil, pkgerrors.NotFound("target pool not found", nil)
	}

	tg := output.TargetGroups[0]
	return &loadbalancer.TargetPool{
		ID:       *tg.TargetGroupArn,
		Name:     *tg.TargetGroupName,
		Protocol: loadbalancer.Protocol(tg.Protocol),
		Port:     int(*tg.Port),
	}, nil
}

func (m *Manager) DeleteTargetPool(ctx context.Context, poolID string) error {
	_, err := m.client.DeleteTargetGroup(ctx, &elasticloadbalancingv2.DeleteTargetGroupInput{
		TargetGroupArn: aws.String(poolID),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete target group", err)
	}
	return nil
}

func (m *Manager) AddTarget(ctx context.Context, poolID string, target loadbalancer.Target) error {
	_, err := m.client.RegisterTargets(ctx, &elasticloadbalancingv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(poolID),
		Targets: []types.TargetDescription{
			{
				Id:   aws.String(target.ID),
				Port: aws.Int32(int32(target.Port)),
			},
		},
	})
	if err != nil {
		return pkgerrors.Internal("failed to register target", err)
	}
	return nil
}

func (m *Manager) RemoveTarget(ctx context.Context, poolID, targetID string) error {
	_, err := m.client.DeregisterTargets(ctx, &elasticloadbalancingv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(poolID),
		Targets: []types.TargetDescription{
			{Id: aws.String(targetID)},
		},
	})
	if err != nil {
		return pkgerrors.Internal("failed to deregister target", err)
	}
	return nil
}

func (m *Manager) GetTargetHealth(ctx context.Context, poolID string) ([]*loadbalancer.Target, error) {
	output, err := m.client.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(poolID),
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to get target health", err)
	}

	result := make([]*loadbalancer.Target, len(output.TargetHealthDescriptions))
	for i, th := range output.TargetHealthDescriptions {
		status := loadbalancer.TargetStatusUnhealthy
		if th.TargetHealth != nil && th.TargetHealth.State == types.TargetHealthStateEnumHealthy {
			status = loadbalancer.TargetStatusHealthy
		}
		result[i] = &loadbalancer.Target{
			ID:     *th.Target.Id,
			Port:   int(*th.Target.Port),
			Status: status,
		}
		if th.TargetHealth != nil && th.TargetHealth.Description != nil {
			result[i].Reason = *th.TargetHealth.Description
		}
	}

	return result, nil
}

func (m *Manager) AddRule(ctx context.Context, listenerID string, rule loadbalancer.Rule) (*loadbalancer.Rule, error) {
	conditions := make([]types.RuleCondition, 0)
	for _, c := range rule.Conditions {
		conditions = append(conditions, types.RuleCondition{
			Field:  aws.String(c.Field),
			Values: c.Values,
		})
	}

	output, err := m.client.CreateRule(ctx, &elasticloadbalancingv2.CreateRuleInput{
		ListenerArn: aws.String(listenerID),
		Priority:    aws.Int32(int32(rule.Priority)),
		Conditions:  conditions,
		Actions: []types.Action{
			{
				Type:           types.ActionTypeEnumForward,
				TargetGroupArn: aws.String(rule.TargetPoolID),
			},
		},
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create rule", err)
	}

	if len(output.Rules) == 0 {
		return nil, pkgerrors.Internal("no rule returned", nil)
	}

	r := output.Rules[0]
	priority := rule.Priority // Use passed priority since SDK returns string
	return &loadbalancer.Rule{
		ID:           *r.RuleArn,
		Priority:     priority,
		TargetPoolID: rule.TargetPoolID,
	}, nil
}

func (m *Manager) RemoveRule(ctx context.Context, listenerID, ruleID string) error {
	_, err := m.client.DeleteRule(ctx, &elasticloadbalancingv2.DeleteRuleInput{
		RuleArn: aws.String(ruleID),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete rule", err)
	}
	return nil
}

// Interface compliance
var _ loadbalancer.LoadBalancerManager = (*Manager)(nil)
