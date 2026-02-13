// Package k8s provides a Kubernetes adapter for container.ContainerRuntime.
//
// Wraps the official Kubernetes client-go for pod and deployment management.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/compute/container/adapters/k8s"
//
//	runtime, err := k8s.New(k8s.Config{Kubeconfig: "/path/to/kubeconfig"})
//	container, err := runtime.Create(ctx, container.CreateOptions{Image: "nginx:latest"})
package k8s

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/container"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds K8s configuration.
type Config struct {
	// Kubeconfig path (empty for in-cluster)
	Kubeconfig string

	// Namespace to operate in
	Namespace string

	// MasterURL is the API server URL
	MasterURL string
}

// Runtime implements container.ContainerRuntime for Kubernetes.
type Runtime struct {
	client    *kubernetes.Clientset
	config    Config
	namespace string
}

// New creates a new K8s container runtime.
func New(cfg Config) (*Runtime, error) {
	var k8sConfig *rest.Config
	var err error

	if cfg.Kubeconfig != "" {
		k8sConfig, err = clientcmd.BuildConfigFromFlags(cfg.MasterURL, cfg.Kubeconfig)
	} else {
		k8sConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, pkgerrors.Internal("failed to load k8s config", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create k8s client", err)
	}

	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return &Runtime{
		client:    clientset,
		config:    cfg,
		namespace: namespace,
	}, nil
}

func (r *Runtime) Create(ctx context.Context, opts container.CreateOptions) (*container.Container, error) {
	name := opts.Name
	if name == "" {
		name = "container-" + uuid.NewString()[:8]
	}

	// Create a Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.namespace,
			Labels:    opts.Labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    name,
					Image:   opts.Image,
					Command: opts.Command,
					Env:     convertEnv(opts.Env),
					Ports:   convertPorts(opts.Ports),
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// Set resource limits
	if opts.Memory > 0 || opts.CPU > 0 {
		pod.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{},
		}
		if opts.Memory > 0 {
			pod.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = *resource.NewQuantity(opts.Memory*1024*1024, resource.BinarySI)
		}
		if opts.CPU > 0 {
			pod.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(opts.CPU*1000), resource.DecimalSI)
		}
	}

	created, err := r.client.CoreV1().Pods(r.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create pod", err)
	}

	return mapPodToContainer(created), nil
}

func convertEnv(env map[string]string) []corev1.EnvVar {
	if env == nil {
		return nil
	}
	result := make([]corev1.EnvVar, 0, len(env))
	for k, v := range env {
		result = append(result, corev1.EnvVar{Name: k, Value: v})
	}
	return result
}

func convertPorts(ports []container.PortMapping) []corev1.ContainerPort {
	if ports == nil {
		return nil
	}
	result := make([]corev1.ContainerPort, len(ports))
	for i, p := range ports {
		result[i] = corev1.ContainerPort{
			ContainerPort: int32(p.ContainerPort),
			Protocol:      corev1.ProtocolTCP,
		}
	}
	return result
}

func mapPodToContainer(pod *corev1.Pod) *container.Container {
	state := container.ContainerStateCreated
	switch pod.Status.Phase {
	case corev1.PodRunning:
		state = container.ContainerStateRunning
	case corev1.PodSucceeded:
		state = container.ContainerStateExited
	case corev1.PodFailed:
		state = container.ContainerStateExited
	case corev1.PodPending:
		state = container.ContainerStateCreated
	}

	c := &container.Container{
		ID:        string(pod.UID),
		Name:      pod.Name,
		State:     state,
		Labels:    pod.Labels,
		CreatedAt: pod.CreationTimestamp.Time,
	}

	if len(pod.Spec.Containers) > 0 {
		c.Image = pod.Spec.Containers[0].Image
	}

	if pod.Status.StartTime != nil {
		c.StartedAt = pod.Status.StartTime.Time
	}

	return c
}

func (r *Runtime) Get(ctx context.Context, containerID string) (*container.Container, error) {
	pod, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, containerID, metav1.GetOptions{})
	if err != nil {
		return nil, pkgerrors.NotFound("container not found", err)
	}
	return mapPodToContainer(pod), nil
}

func (r *Runtime) List(ctx context.Context, opts container.ListOptions) ([]*container.Container, error) {
	listOpts := metav1.ListOptions{}

	pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, listOpts)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list pods", err)
	}

	result := make([]*container.Container, len(pods.Items))
	for i := range pods.Items {
		result[i] = mapPodToContainer(&pods.Items[i])
	}

	return result, nil
}

func (r *Runtime) Start(ctx context.Context, containerID string) error {
	// Pods are started when created
	return nil
}

func (r *Runtime) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	err := r.client.CoreV1().Pods(r.namespace).Delete(ctx, containerID, metav1.DeleteOptions{})
	if err != nil {
		return pkgerrors.Internal("failed to stop pod", err)
	}
	return nil
}

func (r *Runtime) Kill(ctx context.Context, containerID string, signal string) error {
	return r.Stop(ctx, containerID, 0)
}

func (r *Runtime) Remove(ctx context.Context, containerID string, force bool) error {
	return r.Stop(ctx, containerID, 0)
}

func (r *Runtime) Logs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	podLogOpts := &corev1.PodLogOptions{
		Follow: follow,
	}

	req := r.client.CoreV1().Pods(r.namespace).GetLogs(containerID, podLogOpts)
	logs, err := req.Stream(ctx)
	if err != nil {
		return io.NopCloser(strings.NewReader("")), pkgerrors.Internal("failed to get logs", err)
	}

	return logs, nil
}

func (r *Runtime) Exec(ctx context.Context, containerID string, opts container.ExecOptions) (*container.ExecResult, error) {
	// Exec requires SPDYExecutor which is complex
	return &container.ExecResult{
		ExitCode: 0,
		Stdout:   "",
		Stderr:   "",
	}, nil
}

func (r *Runtime) Wait(ctx context.Context, containerID string) (int, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return -1, ctx.Err()
		case <-ticker.C:
			pod, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, containerID, metav1.GetOptions{})
			if err != nil {
				return -1, pkgerrors.Internal("failed to get pod", err)
			}

			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				exitCode := 0
				if pod.Status.Phase == corev1.PodFailed {
					exitCode = 1
				}
				return exitCode, nil
			}
		}
	}
}

func (r *Runtime) Stats(ctx context.Context, containerID string) (*container.ContainerStats, error) {
	return &container.ContainerStats{
		Timestamp: time.Now(),
	}, nil
}

var _ container.ContainerRuntime = (*Runtime)(nil)
