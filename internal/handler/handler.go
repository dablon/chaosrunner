package handler

import (
	"fmt"
	"os"

	"github.com/dablon/chaosrunner/internal/client"
	"github.com/dablon/chaosrunner/internal/experiments"
)

type Handler struct {
	k8sClient *client.K8sClient
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) initK8s() error {
	if h.k8sClient == nil {
		h.k8sClient = client.New()
	}
	if h.k8sClient.Clientset == nil {
		if err := h.k8sClient.Init(); err != nil {
			return fmt.Errorf("failed to initialize k8s client: %v", err)
		}
	}
	return nil
}

func (h *Handler) GetExperiment(name string) (experiments.Experiment, error) {
	if err := h.initK8s(); err != nil {
		return nil, err
	}

	switch name {
	case "pod-kill":
		return experiments.NewPodKillExperiment(h.k8sClient), nil
	case "network-latency":
		return experiments.NewNetworkLatencyExperiment(h.k8sClient), nil
	case "cpu-stress":
		return experiments.NewCpuStressExperiment(h.k8sClient), nil
	case "memory-hog":
		return experiments.NewMemoryHogExperiment(h.k8sClient), nil
	case "disk-fill":
		return experiments.NewDiskFillExperiment(h.k8sClient), nil
	default:
		return nil, fmt.Errorf("unknown experiment: %s", name)
	}
}

func (h *Handler) PodKill(namespace, duration string) error {
	exp, err := h.GetExperiment("pod-kill")
	if err != nil {
		return err
	}
	return exp.Run(namespace, duration)
}

func (h *Handler) NetworkLatency(namespace, duration string) error {
	exp, err := h.GetExperiment("network-latency")
	if err != nil {
		return err
	}
	return exp.Run(namespace, duration)
}

func (h *Handler) CpuStress(namespace, duration string) error {
	exp, err := h.GetExperiment("cpu-stress")
	if err != nil {
		return err
	}
	return exp.Run(namespace, duration)
}

func (h *Handler) MemoryHog(namespace, duration string) error {
	exp, err := h.GetExperiment("memory-hog")
	if err != nil {
		return err
	}
	return exp.Run(namespace, duration)
}

func (h *Handler) DiskFill(namespace, duration string) error {
	exp, err := h.GetExperiment("disk-fill")
	if err != nil {
		return err
	}
	return exp.Run(namespace, duration)
}

func init() {
	os.Setenv("KUBECONFIG", "")
}
