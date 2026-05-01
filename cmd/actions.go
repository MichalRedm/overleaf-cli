package cmd

import (
	"fmt"

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
		if err := client.Compile(); err != nil {
			fmt.Printf("Error during compilation: %v\n", err)
		}
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
		out, err := cmd.Flags().GetString("out")
		if err != nil {
			out = "output.pdf"
		}
		if err := client.DownloadPDF(out); err != nil {
			fmt.Printf("Error downloading PDF: %v\n", err)
		}
	},
}

func init() {
	pdfCmd.Flags().StringP("out", "o", "output.pdf", "output path for PDF")
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(pdfCmd)
}
