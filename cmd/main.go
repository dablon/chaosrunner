package main

import "github.com/spf13/cobra"

func main() {
    root := &cobra.Command{Use: "chaosrunner", Short: "chaosrunner v1.0.0", Run: func(cmd *cobra.Command, args []string) {}}
    root.Execute()
}
