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

func (h *Handler) DryRun(name, namespace, selector string, opts *experiments.ExperimentOptions) error {
	if err := h.initK8s(); err != nil {
		return fmt.Errorf("failed to initialize k8s client: %v", err)
	}

	base := experiments.BaseExperiment{}
	base.PrintDryRunHeader(name, namespace, selector, opts)

	check, err := h.k8sClient.CheckPermissions(opts.Ctx, namespace, selector, name)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %v", err)
	}

	success := true
	if check.CanListPods {
		base.PrintDryRunResult(true, "Can list pods in namespace", opts)
	} else {
		base.PrintDryRunResult(false, "Cannot list pods in namespace", opts)
		success = false
	}

	if check.CanGetPod {
		base.PrintDryRunResult(true, "Can get pods", opts)
	} else {
		base.PrintDryRunResult(false, "Cannot get pods", opts)
		success = false
	}

	if name == "pod-kill" {
		if check.CanDeletePod {
			base.PrintDryRunResult(true, "Can delete pods", opts)
		} else {
			base.PrintDryRunResult(false, "Cannot delete pods", opts)
			success = false
		}
	}

	if name == "cpu-stress" || name == "memory-hog" || name == "network-latency" || name == "disk-fill" {
		if check.CanExecPod {
			base.PrintDryRunResult(true, "Can exec into pods", opts)
		} else {
			base.PrintDryRunResult(false, "Cannot exec into pods", opts)
			success = false
		}
	}

	if selector != "" {
		pods, err := h.k8sClient.GetPodNames(opts.Ctx, namespace, selector)
		if err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}
		if len(pods) > 0 {
			base.PrintDryRunResult(true, fmt.Sprintf("Found %d pods matching selector", len(pods)), opts)
		} else {
			base.PrintDryRunResult(false, "No pods found matching selector", opts)
			success = false
		}
	}

	if !success {
		return fmt.Errorf("permission check failed - see errors above")
	}

	return nil
}
