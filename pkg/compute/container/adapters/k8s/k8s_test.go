package k8s

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestMapPodToContainerUsesNameAsID(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "nginx-abc",
			UID:               types.UID("uid-should-not-be-id"),
			CreationTimestamp: metav1.NewTime(time.Now()),
			Labels:            map[string]string{"app": "web"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "nginx-abc", Image: "nginx:latest"}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	c := mapPodToContainer(pod)
	if c.ID != "nginx-abc" {
		t.Fatalf("expected ID to be pod name, got %q (UID was used historically and broke Get)", c.ID)
	}
	if c.Name != "nginx-abc" {
		t.Fatalf("expected Name %q, got %q", "nginx-abc", c.Name)
	}
	if c.Image != "nginx:latest" {
		t.Fatalf("expected image nginx:latest, got %q", c.Image)
	}
}
