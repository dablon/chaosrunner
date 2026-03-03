package client

import (
	"testing"
)

func TestValidateK8sName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-pod", false},
		{"valid with numbers", "pod123", false},
		{"valid complex", "my-app-123", false},
		{"empty string", "", true},
		{"too long", "a" + string(rune(253)), true},
		{"invalid uppercase", "MyPod", true},
		{"invalid special chars", "my_pod", true},
		{"starts with dash", "-pod", true},
		{"ends with dash", "pod-", true},
		{"has spaces", "my pod", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateK8sName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateK8sName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestK8sClientNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Error("New() should return non-nil K8sClient")
	}
}

func TestK8sPodCaptureResources(t *testing.T) {
	pod := &K8sPod{
		Name:      "test-pod",
		Namespace: "test-ns",
		Phase:     "Running",
		Ready:     true,
		Restarts:  0,
	}

	if pod.Name != "test-pod" {
		t.Errorf("Expected Name 'test-pod', got '%s'", pod.Name)
	}
	if !pod.Ready {
		t.Error("Expected Ready to be true")
	}
}

func TestPodStatsFields(t *testing.T) {
	stats := &PodStats{
		Name:            "test-pod",
		Namespace:       "test-ns",
		Phase:           "Running",
		Ready:           true,
		Restarts:        2,
		TotalContainers: 1,
	}

	if stats.Ready != true {
		t.Error("Expected Ready to be true")
	}
	if stats.Restarts != 2 {
		t.Errorf("Expected Restarts 2, got %d", stats.Restarts)
	}
}
