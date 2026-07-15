package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/loadbalancer"
	lbmem "github.com/chris-alexander-pop/go-hyperforge/pkg/network/loadbalancer/adapters/memory"
)

func TestResilientLoadBalancer_CreateGet(t *testing.T) {
	mgr := loadbalancer.NewResilientManager(lbmem.New(), loadbalancer.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	lb, err := mgr.CreateLoadBalancer(t.Context(), loadbalancer.CreateLoadBalancerOptions{
		Name: "api",
		Type: "application",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := mgr.GetLoadBalancer(t.Context(), lb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "api" {
		t.Fatalf("name=%s", got.Name)
	}
}
