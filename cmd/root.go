package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	configPath := resolveConfigPath(cmd)
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Error loading config from %s: %v\n", configPath, err)
		return nil, nil
	}

	client, err := overleaf.NewClient(cfg.BaseURL, cfg.ProjectID, cfg.Cookie, cfg.AuthType, cfg.AuthCommand, cfg.UseDocker)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return nil, nil
	}

	// Check if authenticated
	if !client.IsAuthenticated() {
		if (cfg.Email != "" && cfg.Password != "") || (cfg.AuthType == "custom" && cfg.AuthCommand != "") {
			fmt.Println("Session expired or invalid, attempting automatic login...")
			if err := client.Login(cfg.Email, cfg.Password); err != nil {
				fmt.Printf("Auto-login failed: %v\n", err)
				return nil, nil
			}
			// Update config with new cookie
			cfg.Cookie = client.Cookie
			if err := config.Save(configPath, cfg); err != nil {
				fmt.Printf("Warning: failed to save updated cookie to config: %v\n", err)
			}
		} else {
			fmt.Println("Session invalid and no credentials/custom auth provided in config. Please run 'init' or update config.")
			return nil, nil
		}
	}

	return client, cfg
}

func resolveConfigPath(cmd *cobra.Command) string {
	configPath, _ := cmd.Flags().GetString("config")

	// If the config flag was not explicitly set by the user
	if !cmd.Flags().Changed("config") {
		src := resolveSrcPath(cmd)
		if src != "" {
			// Try relative to src
			altPath := filepath.Join(src, config.GetConfigPath())
			if _, err := os.Stat(altPath); err == nil {
				return altPath
			}
			// Also check for legacy config in src
			legacyAltPath := filepath.Join(src, config.LegacyConfigFile)
			if _, err := os.Stat(legacyAltPath); err == nil {
				return legacyAltPath
			}
		}

		// Fallback to current directory if src didn't yield anything or wasn't found
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Check for legacy config in current directory as fallback
			if _, err := os.Stat(config.LegacyConfigFile); err == nil {
				return config.LegacyConfigFile
			}
		}
	}

	return configPath
}

func resolveSrcPath(cmd *cobra.Command) string {
	srcFlag := cmd.Flags().Lookup("src")
	if srcFlag == nil {
		// If command doesn't have src flag, still try to find project root from CWD
		if root, err := config.FindProjectRoot("."); err == nil {
			return root
		}
		return "."
	}

	src := srcFlag.Value.String()

	// If src is default (.) and not explicitly changed, try to autodetect
	if !cmd.Flags().Changed("src") {
		if root, err := config.FindProjectRoot("."); err == nil {
			return root
		}
	}

	return src
}


func init() {
	// Global flags can be defined here
	rootCmd.PersistentFlags().StringP("config", "c", config.GetConfigPath(), "path to config file")
}
