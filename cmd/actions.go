package cmd

import (
	"fmt"
	"overleaf-cli/internal/config"
	"overleaf-cli/internal/overleaf"

	"github.com/spf13/cobra"
)

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Trigger project compilation",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient(cmd)
		if client == nil {
			return
		}
		client.Compile()
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show compilation logs",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient(cmd)
		if client == nil {
			return
		}
		client.ShowLogs()
	},
}

var pdfCmd = &cobra.Command{
	Use:   "pdf",
	Short: "Download the compiled PDF",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient(cmd)
		if client == nil {
			return
		}
		out, _ := cmd.Flags().GetString("out")
		client.DownloadPDF(out)
	},
}

func getClient(cmd *cobra.Command) *overleaf.Client {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return nil
	}
	client, err := overleaf.NewClient(cfg.BaseURL, cfg.ProjectID, cfg.Cookie)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return nil
	}
	return client
}

func init() {
	pdfCmd.Flags().StringP("out", "o", "output.pdf", "output path for PDF")
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(pdfCmd)
}
