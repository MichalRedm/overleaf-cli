package cmd

import (
	"github.com/spf13/cobra"
)

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Trigger project compilation",
	Run: func(cmd *cobra.Command, args []string) {
		client, _ := getClient(cmd)
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
		client, _ := getClient(cmd)
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
		client, _ := getClient(cmd)
		if client == nil {
			return
		}
		out, _ := cmd.Flags().GetString("out")
		client.DownloadPDF(out)
	},
}

func init() {
	pdfCmd.Flags().StringP("out", "o", "output.pdf", "output path for PDF")
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(pdfCmd)
}
