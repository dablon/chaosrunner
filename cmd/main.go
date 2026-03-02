package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

var version = "1.0.0"

var experiments = []string{"pod-kill", "network-latency", "cpu-stress", "memory-hog", "disk-fill"}

var runCmd = &cobra.Command{
	Use:   "run [experiment]",
	Short: "Run chaos experiment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		exp := args[0]
		fmt.Printf("Running chaos experiment: %s\n", exp)
		fmt.Println("   Experiment started successfully")
	},
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
