// Package k8s provides a Kubernetes adapter for container.ContainerRuntime.
//
// Pod name is used as the container ID so Create's returned ID works with Get,
// Stop, Logs, and other name-based Kubernetes API calls. The pod UID is kept
// in the hyperforge.io/uid label when present.
package k8s

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/container"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

const uidLabel = "hyperforge.io/uid"

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
	client     *kubernetes.Clientset
	restConfig *rest.Config
	config     Config
	namespace  string
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
		client:     clientset,
		restConfig: k8sConfig,
		config:     cfg,
		namespace:  namespace,
	}, nil
}

func (r *Runtime) Create(ctx context.Context, opts container.CreateOptions) (*container.Container, error) {
	name := opts.Name
	if name == "" {
		name = "container-" + uuid.NewString()[:8]
	}

	labels := map[string]string{}
	for k, v := range opts.Labels {
		labels[k] = v
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.namespace,
			Labels:    labels,
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

	// Persist UID on the object for callers that need it, without using it as ID.
	if created.Labels == nil {
		created.Labels = map[string]string{}
	}
	created.Labels[uidLabel] = string(created.UID)
	_, _ = r.client.CoreV1().Pods(r.namespace).Update(ctx, created, metav1.UpdateOptions{})

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

// mapPodToContainer maps a Pod to a Container.
// ID is the pod name so Create → Get round-trips via the Kubernetes name API.
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
		ID:        pod.Name,
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

// resolvePodName returns the pod name for API calls.
// Accepts the Create-returned ID (pod name) or a legacy UID string.
func (r *Runtime) resolvePodName(ctx context.Context, containerID string) (string, error) {
	_, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, containerID, metav1.GetOptions{})
	if err == nil {
		return containerID, nil
	}
	if !apierrors.IsNotFound(err) {
		return "", pkgerrors.Internal("failed to get pod", err)
	}

	// Legacy: callers may still pass a UID from older Create responses.
	list, listErr := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: uidLabel + "=" + containerID,
	})
	if listErr != nil {
		return "", pkgerrors.Internal("failed to list pods by uid", listErr)
	}
	if len(list.Items) == 1 {
		return list.Items[0].Name, nil
	}

	// Field selector on metadata.uid (supported by kube-apiserver).
	list, listErr = r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "metadata.uid=" + containerID,
	})
	if listErr != nil {
		return "", container.ErrContainerNotFound
	}
	if len(list.Items) == 1 {
		return list.Items[0].Name, nil
	}

	return "", container.ErrContainerNotFound
}

func (r *Runtime) Get(ctx context.Context, containerID string) (*container.Container, error) {
	name, err := r.resolvePodName(ctx, containerID)
	if err != nil {
		return nil, err
	}
	pod, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, container.ErrContainerNotFound
		}
		return nil, pkgerrors.Internal("failed to get pod", err)
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
	// Pods are started when created.
	_, err := r.resolvePodName(ctx, containerID)
	return err
}

func (r *Runtime) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	name, err := r.resolvePodName(ctx, containerID)
	if err != nil {
		return err
	}
	err = r.client.CoreV1().Pods(r.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return container.ErrContainerNotFound
		}
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
	name, err := r.resolvePodName(ctx, containerID)
	if err != nil {
		return nil, err
	}

	podLogOpts := &corev1.PodLogOptions{
		Follow: follow,
	}

	req := r.client.CoreV1().Pods(r.namespace).GetLogs(name, podLogOpts)
	logs, err := req.Stream(ctx)
	if err != nil {
		return io.NopCloser(strings.NewReader("")), pkgerrors.Internal("failed to get logs", err)
	}

	return logs, nil
}

func (r *Runtime) Exec(ctx context.Context, containerID string, opts container.ExecOptions) (*container.ExecResult, error) {
	name, err := r.resolvePodName(ctx, containerID)
	if err != nil {
		return nil, err
	}
	if len(opts.Command) == 0 {
		return nil, pkgerrors.InvalidArgument("command is required", nil)
	}
	if r.restConfig == nil {
		return nil, pkgerrors.Unimplemented("k8s exec requires a REST config (use New)", nil)
	}

	pod, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, container.ErrContainerNotFound
		}
		return nil, pkgerrors.Internal("failed to get pod for exec", err)
	}
	containerName := name
	if len(pod.Spec.Containers) > 0 {
		containerName = pod.Spec.Containers[0].Name
	}

	req := r.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(r.namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   opts.Command,
		Stdout:    true,
		Stderr:    true,
		TTY:       opts.Tty,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(r.restConfig, "POST", req.URL())
	if err != nil {
		return nil, pkgerrors.Internal("failed to create SPDY executor", err)
	}

	var stdout, stderr strings.Builder
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    opts.Tty,
	})
	exitCode := 0
	if err != nil {
		exitCode = 1
		if ee, ok := err.(interface{ ExitStatus() int }); ok {
			exitCode = ee.ExitStatus()
		} else if !strings.Contains(err.Error(), "command terminated with exit code") {
			return &container.ExecResult{
				ExitCode: exitCode,
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
			}, pkgerrors.Internal("pod exec failed", err)
		}
	}
	return &container.ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func (r *Runtime) Wait(ctx context.Context, containerID string) (int, error) {
	name, err := r.resolvePodName(ctx, containerID)
	if err != nil {
		return -1, err
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return -1, ctx.Err()
		case <-ticker.C:
			pod, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, name, metav1.GetOptions{})
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
	if _, err := r.resolvePodName(ctx, containerID); err != nil {
		return nil, err
	}
	// Metrics require metrics.k8s.io (metrics-server). Return a clear Unimplemented
	// rather than an empty stub that looks successful.
	return nil, pkgerrors.Unimplemented("k8s container Stats requires metrics-server (metrics.k8s.io); not wired", nil)
}

var _ container.ContainerRuntime = (*Runtime)(nil)
