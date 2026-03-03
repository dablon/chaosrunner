package main

import (
	"fmt"
	"os"

	"github.com/dablon/chaosrunner/internal/handler"
	"github.com/spf13/cobra"
)

var version = "1.0.0"

var experiments = []string{"pod-kill", "network-latency", "cpu-stress", "memory-hog", "disk-fill"}

var namespace string
var duration string

var runCmd = &cobra.Command{
	Use:   "run [experiment]",
	Short: "Run chaos experiment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		exp := args[0]

		h := handler.New()

		var err error
		switch exp {
		case "pod-kill":
			err = h.PodKill(namespace, duration)
		case "network-latency":
			err = h.NetworkLatency(namespace, duration)
		case "cpu-stress":
			err = h.CpuStress(namespace, duration)
		case "memory-hog":
			err = h.MemoryHog(namespace, duration)
		case "disk-fill":
			err = h.DiskFill(namespace, duration)
		default:
			fmt.Printf("Error: unknown experiment '%s'\n", exp)
			fmt.Println("Use 'chaosrunner list' to see available experiments")
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	runCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Target namespace")
	runCmd.Flags().StringVarP(&duration, "duration", "d", "5m", "Experiment duration")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available experiments",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available experiments:")
		for _, e := range experiments {
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
	root.AddCommand(listCmd)
	root.AddCommand(versionCmd)
	root.Execute()
}
