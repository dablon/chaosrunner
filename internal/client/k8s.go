package client

import (
	"context"
	"fmt"
	"os"

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
			return &K8sPod{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			}, nil
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
	for _, pod := range pods.Items {
		result = append(result, K8sPod{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
		})
	}
	return result, nil
}

func (c *K8sClient) DeletePod(namespace, name string) error {
	ctx := context.Background()
	return c.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

type K8sPod struct {
	Name      string
	Namespace string
	Phase     string
}
