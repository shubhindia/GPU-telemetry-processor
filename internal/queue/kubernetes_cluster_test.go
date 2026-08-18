package queue

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesClusterNodes(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "queue-0",
				Namespace: "default",
				Labels: map[string]string{
					"app": "gpu-telemetry-queue",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "queue-1",
				Namespace: "default",
				Labels: map[string]string{
					"app": "gpu-telemetry-queue",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other",
				Namespace: "default",
				Labels: map[string]string{
					"app": "something-else",
				},
			},
		},
	)

	cluster := NewKubernetesCluster(
		clientset,
		"default",
		"app=gpu-telemetry-queue",
		"queue",
		8080,
	)

	nodes, err := cluster.Nodes(context.Background())
	if err != nil {
		t.Fatalf("discover nodes: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	if nodes[0].ID != "queue-0" {
		t.Fatalf("expected queue-0, got %q", nodes[0].ID)
	}

	if nodes[0].Address != "http://queue-0.queue.default.svc:8080" {
		t.Fatalf("unexpected address %q", nodes[0].Address)
	}
}
