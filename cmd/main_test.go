package main

import (
	"testing"
)

func TestVersionValue(t *testing.T) {
	if version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version)
	}
}

func TestExperimentList(t *testing.T) {
	expected := []string{"pod-kill", "network-latency", "cpu-stress", "memory-hog", "disk-fill"}

	if len(experimentList) != len(expected) {
		t.Errorf("Expected %d experiments, got %d", len(expected), len(experimentList))
	}

	for i, exp := range expected {
		if experimentList[i] != exp {
			t.Errorf("Expected experiment %s at index %d, got %s", exp, i, experimentList[i])
		}
	}
}
