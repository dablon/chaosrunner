package handler

import "testing"

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
