package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dablon/chaosrunner/internal/client"
	"github.com/dablon/chaosrunner/internal/experiments"
	"github.com/dablon/chaosrunner/internal/handler"
	"github.com/spf13/cobra"
)

var version = "1.0.0"

var experimentList = []string{"pod-kill", "network-latency", "cpu-stress", "memory-hog", "disk-fill"}

var namespace string
var duration string
var selector string
var output string
var workers int
var memory string
var delay string
var diskSize string
var allPods bool
var dryRun bool
var webhookURL string
var prometheus bool

var runCmd = &cobra.Command{
	Use:   "run [experiment]",
	Short: "Run chaos experiment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		expName := args[0]

		if err := client.ValidateK8sName(namespace); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid namespace '%s': %v\n", namespace, err)
			os.Exit(1)
		}

		h := handler.New()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		opts := experiments.DefaultOptions(ctx)
		opts.Selector = selector
		opts.Output = output
		opts.Workers = workers
		opts.Memory = memory
		opts.Delay = delay
		opts.DiskSize = diskSize
		opts.AllPods = allPods
		opts.DryRun = dryRun
		opts.WebhookURL = webhookURL
		opts.Prometheus = prometheus

		err := h.RunExperiment(expName, namespace, duration, opts)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var dryRunCmd = &cobra.Command{
	Use:   "dry-run [experiment]",
	Short: "Validate permissions without running experiment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		expName := args[0]

		if err := client.ValidateK8sName(namespace); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid namespace '%s': %v\n", namespace, err)
			os.Exit(1)
		}

		h := handler.New()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		opts := experiments.DefaultOptions(ctx)
		opts.Selector = selector

		err := h.DryRun(expName, namespace, selector, opts)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	runCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Target namespace")
	runCmd.Flags().StringVarP(&duration, "duration", "d", "5m", "Experiment duration (e.g., 30s, 5m, 1h)")
	runCmd.Flags().StringVarP(&selector, "selector", "l", "", "Label selector for pod targeting (e.g., app=nginx)")
	runCmd.Flags().StringVarP(&output, "output", "o", "text", "Output format: text, json, or prometheus")
	runCmd.Flags().IntVarP(&workers, "workers", "w", 4, "Number of CPU stress workers (cpu-stress experiment)")
	runCmd.Flags().StringVarP(&memory, "memory", "m", "256M", "Memory size for memory-hog experiment (e.g., 256M, 512M)")
	runCmd.Flags().StringVarP(&delay, "delay", "", "100ms", "Network latency delay (e.g., 100ms, 1s, 5s)")
	runCmd.Flags().StringVarP(&diskSize, "size", "s", "100M", "Disk fill size per iteration (e.g., 100M, 500M)")
	runCmd.Flags().BoolVar(&allPods, "all-pods", false, "Run experiment on all matching pods (not just first one)")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate permissions without running experiment")
	runCmd.Flags().StringVar(&webhookURL, "webhook", "", "URL to send webhook notification after experiment")
	runCmd.Flags().BoolVar(&prometheus, "prometheus", false, "Output metrics in Prometheus format")

	dryRunCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Target namespace")
	dryRunCmd.Flags().StringVarP(&selector, "selector", "l", "", "Label selector for pod targeting (e.g., app=nginx)")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available experiments",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available experiments:")
		for _, e := range experimentList {
			fmt.Printf("   - %s\n", e)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func main() {
	root := &cobra.Command{
		Use:   "chaosrunner",
		Short: "Chaos Engineering Tool",
	}
	root.AddCommand(runCmd)
	root.AddCommand(dryRunCmd)
	root.AddCommand(listCmd)
	root.AddCommand(versionCmd)
	root.Execute()
}
