package client

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var validK8sNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func ValidateK8sName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 253 {
		return fmt.Errorf("name cannot exceed 253 characters")
	}
	if !validK8sNameRegex.MatchString(name) {
		return fmt.Errorf("name must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character")
	}
	return nil
}

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

func (c *K8sClient) GetRunningPod(ctx context.Context, namespace, labelSelector string) (*K8sPod, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
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
				Ready:     isPodReady(&pod),
			}
			k8sPod.CaptureResources(&pod)
			return k8sPod, nil
		}
	}
	return nil, fmt.Errorf("no running pods found in namespace %s", namespace)
}

func (c *K8sClient) GetPods(ctx context.Context, namespace, labelSelector string) ([]K8sPod, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
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
			Ready:     isPodReady(pod),
		}
		k8sPod.CaptureResources(pod)
		result = append(result, k8sPod)
	}
	return result, nil
}

func (c *K8sClient) DeletePod(ctx context.Context, namespace, name string) error {
	return c.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *K8sClient) WaitForPodReady(ctx context.Context, namespace, podName string, timeout time.Duration) error {
	start := time.Now()

	for time.Since(start) < timeout {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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

func (c *K8sClient) WaitForNewPodReady(ctx context.Context, namespace, labelSelector, excludePodName string, timeout time.Duration) error {
	start := time.Now()

	for time.Since(start) < timeout {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.Name != excludePodName && pod.Status.Phase == "Running" && isPodReady(pod) {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for new pod to be ready in namespace %s", namespace)
}

func (c *K8sClient) GetPodStats(ctx context.Context, namespace, podName string) (*PodStats, error) {
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	stats := &PodStats{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Phase:     string(pod.Status.Phase),
		Age:       time.Since(pod.CreationTimestamp.Time).Round(time.Second),
		Ready:     isPodReady(pod),
	}

	var totalRestarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Running != nil {
			stats.StartedAt = cs.State.Running.StartedAt.Time
		}
		totalRestarts += cs.RestartCount
	}

	stats.Restarts = totalRestarts
	stats.TotalContainers = len(pod.Status.ContainerStatuses)

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
	Ready         bool
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

	p.Ready = isPodReady(pod)
	var totalRestarts int32
	for _, c := range pod.Status.ContainerStatuses {
		totalRestarts += c.RestartCount
	}
	p.Restarts = totalRestarts
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
