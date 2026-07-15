package kubernetes_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
	k8sdisco "github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery/adapters/kubernetes"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
)

func TestKubernetesRegistry_EndpointsCRUD(t *testing.T) {
	client := fake.NewSimpleClientset()
	reg, err := k8sdisco.New(k8sdisco.Config{
		Namespace: "default",
		Client:    client,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = reg.Close() })

	ctx := context.Background()
	svc, err := reg.Register(ctx, discovery.RegisterOptions{
		Name:    "api",
		Address: "10.0.0.5",
		Port:    8080,
		Tags:    []string{"v1"},
		Weight:  3,
	})
	require.NoError(t, err)
	require.Equal(t, "api", svc.Name)
	require.NotEmpty(t, svc.ID)

	got, err := reg.Get(ctx, svc.ID)
	require.NoError(t, err)
	require.Equal(t, "10.0.0.5", got.Address)
	require.Equal(t, 3, got.Weight)

	list, err := reg.Lookup(ctx, "api", discovery.QueryOptions{})
	require.NoError(t, err)
	require.Len(t, list, 1)

	all, err := reg.List(ctx, discovery.QueryOptions{Tag: "v1"})
	require.NoError(t, err)
	require.Len(t, all, 1)

	require.NoError(t, reg.Heartbeat(ctx, svc.ID))
	require.NoError(t, reg.Deregister(ctx, svc.ID))
	_, err = reg.Get(ctx, svc.ID)
	require.ErrorIs(t, err, discovery.ErrServiceNotFound)
}

func TestKubernetesRegistry_EndpointSliceLookup(t *testing.T) {
	client := fake.NewSimpleClientset()
	port := int32(80)
	_, err := client.DiscoveryV1().EndpointSlices("default").Create(context.Background(), &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-abc",
			Namespace: "default",
			Labels:    map[string]string{"kubernetes.io/service-name": "web"},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{{
			Addresses: []string{"10.1.1.1"},
			TargetRef: &corev1.ObjectReference{
				Kind: "Pod",
				Name: "pod-1",
			},
		}},
		Ports: []discoveryv1.EndpointPort{{
			Port: ptr.To(port),
		}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	reg, err := k8sdisco.New(k8sdisco.Config{
		Namespace:           "default",
		Client:              client,
		PreferEndpointSlice: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = reg.Close() })

	list, err := reg.Lookup(context.Background(), "web", discovery.QueryOptions{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "10.1.1.1", list[0].Address)
	require.Equal(t, "pod-1", list[0].ID)
}

func TestKubernetesRegistry_Watch(t *testing.T) {
	client := fake.NewSimpleClientset()
	reg, err := k8sdisco.New(k8sdisco.Config{Namespace: "default", Client: client})
	require.NoError(t, err)
	t.Cleanup(func() { _ = reg.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := reg.Watch(ctx, "watchme")
	require.NoError(t, err)

	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = reg.Register(context.Background(), discovery.RegisterOptions{
			Name: "watchme", Address: "10.9.9.9", Port: 9,
		})
	}()

	seen := false
	for snap := range ch {
		if len(snap) > 0 {
			seen = true
			cancel()
			break
		}
	}
	require.True(t, seen)
}
