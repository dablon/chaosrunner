package handler

import (
	"context"
	"testing"

	"github.com/dablon/chaosrunner/internal/experiments"
)

func TestNew(t *testing.T) {
	h := New()
	if h == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestInitK8sNoClient(t *testing.T) {
	h := New()
	h.k8sClient = nil
	err := h.initK8s()
	if err != nil {
		t.Logf("Expected error when KUBECONFIG not set: %v", err)
	}
}

func TestGetExperiment(t *testing.T) {
	h := New()

	tests := []struct {
		name    string
		expName string
		wantNil bool
	}{
		{"pod-kill", "pod-kill", false},
		{"network-latency", "network-latency", false},
		{"cpu-stress", "cpu-stress", false},
		{"memory-hog", "memory-hog", false},
		{"disk-fill", "disk-fill", false},
		{"unknown", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := h.GetExperiment(tt.expName)
			if tt.wantNil {
				if err == nil {
					t.Error("Expected error for unknown experiment")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if exp == nil {
					t.Error("Expected non-nil experiment")
				}
			}
		})
	}
}

func TestRunExperimentWithoutK8s(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test")
	}
	h := New()
	ctx := context.Background()
	opts := experiments.DefaultOptions(ctx)

	err := h.RunExperiment("pod-kill", "nonexistent-ns", "1s", opts)
	if err == nil {
		t.Error("Expected error when namespace doesn't exist")
	}
}
