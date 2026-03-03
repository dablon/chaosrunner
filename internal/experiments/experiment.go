package experiments

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Experiment is the interface all chaos experiments must implement.
type Experiment interface {
	Name() string
	Run(namespace, duration string, opts *ExperimentOptions) error
}

// ExperimentOptions holds configurable parameters for experiments.
type ExperimentOptions struct {
	Ctx      context.Context
	Selector string // label selector for pod targeting (e.g. "app=nginx")
	Output   string // "text" or "json"
	Workers  int    // cpu-stress worker count
	Memory   string // memory-hog size (e.g. "256M")
	Delay    string // network-latency delay (e.g. "100ms")
	DiskSize string // disk-fill size per iteration (e.g. "100M")
}

// DefaultOptions returns ExperimentOptions with sane defaults.
func DefaultOptions(ctx context.Context) *ExperimentOptions {
	return &ExperimentOptions{
		Ctx:      ctx,
		Selector: "",
		Output:   "text",
		Workers:  4,
		Memory:   "256M",
		Delay:    "100ms",
		DiskSize: "100M",
	}
}

// ExperimentResult holds structured results for JSON output.
type ExperimentResult struct {
	Experiment string                 `json:"experiment"`
	Namespace  string                 `json:"namespace"`
	Duration   string                 `json:"duration"`
	Success    bool                   `json:"success"`
	Metrics    map[string]interface{} `json:"metrics"`
	Error      string                 `json:"error,omitempty"`
}

// PrintJSON marshals and prints the result as JSON to stdout.
func (r *ExperimentResult) PrintJSON() {
	data, _ := json.MarshalIndent(r, "", "  ")
	fmt.Fprintln(os.Stdout, string(data))
}

// BaseExperiment provides shared output helpers.
type BaseExperiment struct{}

func (b *BaseExperiment) PrintHeader(name string, opts *ExperimentOptions) {
	if opts.Output == "json" {
		return
	}
	fmt.Println("🔥 Running chaos experiment:", name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n📋 Experiment: %s\n", name)
}

func (b *BaseExperiment) PrintFooter(duration string, opts *ExperimentOptions) {
	if opts.Output == "json" {
		return
	}
	fmt.Printf("\n✅ Experiment completed successfully\n")
	fmt.Printf("   Duration: %s\n", duration)
}

// IsTextOutput returns true if the output mode is text (not json).
func IsTextOutput(opts *ExperimentOptions) bool {
	return opts.Output != "json"
}

// ParseDuration parses a duration string like "5m", "30s", "1h".
func ParseDuration(d string) (time.Duration, error) {
	duration, err := time.ParseDuration(d)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format '%s'. Use format like 5m, 30s, 1h", d)
	}
	return duration, nil
}
