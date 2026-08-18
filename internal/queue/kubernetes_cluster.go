package queue

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesCluster struct {
	clientset kubernetes.Interface
	namespace string
	selector  string
	service   string
	port      int
}

func NewKubernetesCluster(
	clientset kubernetes.Interface,
	namespace string,
	selector string,
	service string,
	port int,
) *KubernetesCluster {
	return &KubernetesCluster{
		clientset: clientset,
		namespace: namespace,
		selector:  selector,
		service:   service,
		port:      port,
	}
}

func NewInClusterKubernetesCluster(
	namespace string,
	selector string,
	service string,
	port int,
) (*KubernetesCluster, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	return NewKubernetesCluster(
		clientset,
		namespace,
		selector,
		service,
		port,
	), nil
}

func (c *KubernetesCluster) Nodes(
	ctx context.Context,
) ([]Node, error) {
	pods, err := c.clientset.CoreV1().
		Pods(c.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: c.selector,
		})
	if err != nil {
		return nil, fmt.Errorf("list queue pods: %w", err)
	}

	nodes := make([]Node, 0, len(pods.Items))

	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		if !podDiscoverable(pod) {
			continue
		}

		nodes = append(nodes, Node{
			ID: pod.Name,
			Address: fmt.Sprintf(
				"http://%s.%s.%s.svc:%d",
				pod.Name,
				c.service,
				c.namespace,
				c.port,
			),
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	return nodes, nil
}

func podDiscoverable(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning
}
