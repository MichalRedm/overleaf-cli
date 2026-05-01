package cmd

import (
	"fmt"
	"os"

	"overleaf-cli/internal/config"
	"overleaf-cli/internal/overleaf"

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

func getClient(cmd *cobra.Command) (*overleaf.Client, *config.Config) {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return nil, nil
	}

	client, err := overleaf.NewClient(cfg.BaseURL, cfg.ProjectID, cfg.Cookie)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return nil, nil
	}

	// Check if authenticated
	if !client.IsAuthenticated() {
		if cfg.Email != "" && cfg.Password != "" {
			fmt.Println("Session expired or invalid, attempting automatic login...")
			if err := client.Login(cfg.Email, cfg.Password); err != nil {
				fmt.Printf("Auto-login failed: %v\n", err)
				return nil, nil
			}
			// Update config with new cookie
			cfg.Cookie = client.Cookie
			config.Save(configPath, cfg)
		} else {
			fmt.Println("Session invalid and no credentials provided in config. Please run 'init' or update config.")
			return nil, nil
		}
	}

	return client, cfg
}


func init() {
	// Global flags can be defined here
	rootCmd.PersistentFlags().StringP("config", "c", "overleaf_config.json", "path to config file")
}
