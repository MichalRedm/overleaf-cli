package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "overleaf-cli",
	Short: "Overleaf CLI is a tool for synchronizing local directories with Overleaf CE",
	Long:  `A robust CLI utility to manage, sync, and compile your local LaTeX projects with a self-hosted Overleaf (ShareLaTeX) instance.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be defined here
	rootCmd.PersistentFlags().StringP("config", "c", "overleaf_config.json", "path to config file")
}
