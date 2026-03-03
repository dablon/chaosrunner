package experiments

import (
	"context"
	"testing"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid seconds", "30s", false},
		{"valid minutes", "5m", false},
		{"valid hours", "1h", false},
		{"valid compound", "1h30m", false},
		{"invalid format", "abc", true},
		{"empty string", "", true},
		{"invalid unit", "5x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	ctx := context.Background()
	opts := DefaultOptions(ctx)

	if opts.Ctx != ctx {
		t.Error("DefaultOptions should set correct context")
	}
	if opts.Output != "text" {
		t.Errorf("Expected output 'text', got '%s'", opts.Output)
	}
	if opts.Workers != 4 {
		t.Errorf("Expected workers 4, got %d", opts.Workers)
	}
	if opts.Memory != "256M" {
		t.Errorf("Expected memory '256M', got '%s'", opts.Memory)
	}
	if opts.Delay != "100ms" {
		t.Errorf("Expected delay '100ms', got '%s'", opts.Delay)
	}
	if opts.DiskSize != "100M" {
		t.Errorf("Expected diskSize '100M', got '%s'", opts.DiskSize)
	}
}

func TestIsTextOutput(t *testing.T) {
	ctx := context.Background()

	textOpts := DefaultOptions(ctx)
	textOpts.Output = "text"
	if !IsTextOutput(textOpts) {
		t.Error("Expected IsTextOutput to return true for 'text'")
	}

	jsonOpts := DefaultOptions(ctx)
	jsonOpts.Output = "json"
	if IsTextOutput(jsonOpts) {
		t.Error("Expected IsTextOutput to return false for 'json'")
	}
}

func TestExperimentResult(t *testing.T) {
	result := &ExperimentResult{
		Experiment: "pod-kill",
		Namespace:  "test-ns",
		Duration:   "30s",
		Success:    true,
		Metrics:    map[string]interface{}{"iterations": 5},
	}

	result.Error = "test error"

	if result.Experiment != "pod-kill" {
		t.Errorf("Expected experiment 'pod-kill', got '%s'", result.Experiment)
	}
	if result.Error != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", result.Error)
	}
}

func TestPrintHeaderWithJSON(t *testing.T) {
	base := BaseExperiment{}
	ctx := context.Background()
	opts := DefaultOptions(ctx)
	opts.Output = "json"

	base.PrintHeader("pod-kill", opts)
}

func TestPrintFooterWithJSON(t *testing.T) {
	base := BaseExperiment{}
	ctx := context.Background()
	opts := DefaultOptions(ctx)
	opts.Output = "json"

	base.PrintFooter("30s", opts)
}
