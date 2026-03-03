package handler

import (
	"fmt"

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

func (h *Handler) RunExperiment(name, namespace, duration string, opts *experiments.ExperimentOptions) error {
	exp, err := h.GetExperiment(name)
	if err != nil {
		return err
	}
	return exp.Run(namespace, duration, opts)
}
