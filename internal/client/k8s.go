package client

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	Clientset *kubernetes.Clientset
}

func New() *K8sClient {
	return &K8sClient{}
}

func (c *K8sClient) Init() error {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build config: %v", err)
	}

	c.Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %v", err)
	}
	return nil
}

func (c *K8sClient) GetRunningPod(namespace string) (*K8sPod, error) {
	ctx := context.Background()
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range pods.Items {
		pod := pods.Items[i]
		if pod.Status.Phase == "Running" {
			k8sPod := &K8sPod{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Phase:     string(pod.Status.Phase),
			}
			k8sPod.CaptureResources(&pod)
			return k8sPod, nil
		}
	}
	return nil, fmt.Errorf("no running pods found in namespace %s", namespace)
}

func (c *K8sClient) GetPods(namespace string) ([]K8sPod, error) {
	ctx := context.Background()
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]K8sPod, 0, len(pods.Items))
	for i := range pods.Items {
		pod := &pods.Items[i]
		k8sPod := K8sPod{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
		}
		k8sPod.CaptureResources(pod)
		result = append(result, k8sPod)
	}
	return result, nil
}

func (c *K8sClient) DeletePod(namespace, name string) error {
	ctx := context.Background()
	return c.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *K8sClient) WaitForPodReady(namespace, podName string, timeout time.Duration) error {
	ctx := context.Background()
	start := time.Now()

	for time.Since(start) < timeout {
		pod, err := c.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if isPodReady(pod) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for pod %s to be ready", podName)
}

func (c *K8sClient) GetPodStats(namespace, podName string) (*PodStats, error) {
	ctx := context.Background()
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	stats := &PodStats{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Phase:     string(pod.Status.Phase),
		Age:       time.Since(pod.CreationTimestamp.Time).Round(time.Second),
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Running != nil {
			stats.Ready = true
			stats.Restarts = cs.RestartCount
			stats.StartedAt = cs.State.Running.StartedAt.Time
		}
	}

	if len(pod.Status.ContainerStatuses) > 0 {
		stats.TotalContainers = len(pod.Status.ContainerStatuses)
		for _, cs := range pod.Status.ContainerStatuses {
			stats.Restarts += cs.RestartCount
		}
	}

	return stats, nil
}

func isPodReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

type K8sPod struct {
	Name          string
	Namespace     string
	Phase         string
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string
	Ready         string
	Restarts      int32
	Age           time.Duration
}

func (p *K8sPod) CaptureResources(pod *corev1.Pod) {
	for _, c := range pod.Spec.Containers {
		if c.Resources.Requests != nil {
			if cpu, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
				p.CPURequest = cpu.String()
			}
			if mem, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
				p.MemoryRequest = mem.String()
			}
		}
		if c.Resources.Limits != nil {
			if cpu, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
				p.CPULimit = cpu.String()
			}
			if mem, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
				p.MemoryLimit = mem.String()
			}
		}
	}

	for _, c := range pod.Status.ContainerStatuses {
		if c.Ready {
			p.Ready = "1/1"
		}
		p.Restarts = c.RestartCount
	}
	if p.Ready == "" {
		p.Ready = "0/1"
	}
}

type PodStats struct {
	Name            string
	Namespace       string
	Phase           string
	Age             time.Duration
	Ready           bool
	Restarts        int32
	TotalContainers int
	StartedAt       time.Time
}
