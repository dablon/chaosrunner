package experiments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Experiment is the interface all chaos experiments must implement.
type Experiment interface {
	Name() string
	Run(namespace, duration string, opts *ExperimentOptions) error
}

// ExperimentOptions holds configurable parameters for experiments.
type ExperimentOptions struct {
	Ctx        context.Context
	Selector   string // label selector for pod targeting (e.g. "app=nginx")
	Output     string // "text", "json", or "prometheus"
	Workers    int    // cpu-stress worker count
	Memory     string // memory-hog size (e.g. "256M")
	Delay      string // network-latency delay (e.g. "100ms")
	DiskSize   string // disk-fill size per iteration (e.g. "100M")
	AllPods    bool   // run experiment on all matching pods (not just first one)
	DryRun     bool   // validate permissions without running experiment
	WebhookURL string // URL to send webhook notification after experiment
	Prometheus bool   // output metrics in Prometheus format
}

// DefaultOptions returns ExperimentOptions with sane defaults.
func DefaultOptions(ctx context.Context) *ExperimentOptions {
	return &ExperimentOptions{
		Ctx:        ctx,
		Selector:   "",
		Output:     "text",
		Workers:    4,
		Memory:     "256M",
		Delay:      "100ms",
		DiskSize:   "100M",
		AllPods:    false,
		DryRun:     false,
		WebhookURL: "",
		Prometheus: false,
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
	TargetPods []string               `json:"target_pods,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// PrintJSON marshals and prints the result as JSON to stdout.
func (r *ExperimentResult) PrintJSON() {
	r.Timestamp = time.Now()
	data, _ := json.MarshalIndent(r, "", "  ")
	fmt.Fprintln(os.Stdout, string(data))
}

// PrometheusMetrics holds metrics in Prometheus format
type PrometheusMetrics struct {
	Name      string
	Labels    string
	Value     float64
	Timestamp int64
}

// PrintPrometheus outputs metrics in Prometheus exposition format
func PrintPrometheus(metrics []PrometheusMetrics) {
	for _, m := range metrics {
		labels := m.Labels
		if labels != "" {
			labels = "{" + labels + "}"
		}
		fmt.Printf("chaosrunner_%s%s %.2f %d\n", m.Name, labels, m.Value, m.Timestamp)
	}
}

// BaseExperiment provides shared output helpers.
type BaseExperiment struct{}

func (b *BaseExperiment) PrintHeader(name string, opts *ExperimentOptions) {
	if opts.Output == "json" || opts.Output == "prometheus" {
		return
	}
	fmt.Println("🔥 Running chaos experiment:", name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n📋 Experiment: %s\n", name)
}

func (b *BaseExperiment) PrintFooter(duration string, opts *ExperimentOptions) {
	if opts.Output == "json" || opts.Output == "prometheus" {
		return
	}
	fmt.Printf("\n✅ Experiment completed successfully\n")
	fmt.Printf("   Duration: %s\n", duration)
}

func (b *BaseExperiment) PrintDryRunHeader(name, namespace, selector string, opts *ExperimentOptions) {
	if opts.Output == "json" || opts.Output == "prometheus" {
		return
	}
	fmt.Println("🔍 Dry-run mode - validating permissions")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n📋 Experiment: %s\n", name)
	fmt.Printf("   Namespace: %s\n", namespace)
	if selector != "" {
		fmt.Printf("   Selector: %s\n", selector)
	}
}

func (b *BaseExperiment) PrintDryRunResult(success bool, message string, opts *ExperimentOptions) {
	if opts.Output == "json" || opts.Output == "prometheus" {
		return
	}
	if success {
		fmt.Printf("   ✅ %s\n", message)
	} else {
		fmt.Printf("   ❌ %s\n", message)
	}
}

// IsTextOutput returns true if the output mode is text (not json).
func IsTextOutput(opts *ExperimentOptions) bool {
	return opts.Output != "json" && opts.Output != "prometheus"
}

// IsJSONOutput returns true if the output mode is json.
func IsJSONOutput(opts *ExperimentOptions) bool {
	return opts.Output == "json"
}

// IsPrometheusOutput returns true if the output mode is prometheus.
func IsPrometheusOutput(opts *ExperimentOptions) bool {
	return opts.Output == "prometheus" || opts.Prometheus
}

// ParseDuration parses a duration string like "5m", "30s", "1h".
func ParseDuration(d string) (time.Duration, error) {
	duration, err := time.ParseDuration(d)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format '%s'. Use format like 5m, 30s, 1h", d)
	}
	return duration, nil
}

// SendWebhook sends experiment result to configured webhook URL
func SendWebhook(webhookURL string, result *ExperimentResult) error {
	if webhookURL == "" {
		return nil
	}

	result.Timestamp = time.Now()

	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", webhookURL, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	}

	return nil
}
