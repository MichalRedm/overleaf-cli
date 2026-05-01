package cmd

import (
	"bufio"
	"fmt"
	"os"
	"overleaf-cli/internal/config"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration for Overleaf CLI",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			configPath = "overleaf_config.json"
		}
		
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter Overleaf Base URL (e.g., http://localhost:80): ")
		baseURL, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		baseURL = strings.TrimSpace(baseURL)

		fmt.Print("Enter overleaf.sid cookie value (optional, press enter to skip and use email/pass): ")
		cookie, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		cookie = strings.TrimSpace(cookie)

		fmt.Print("Enter Email: ")
		email, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		email = strings.TrimSpace(email)

		fmt.Print("Enter Password: ")
		password, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		password = strings.TrimSpace(password)

		fmt.Print("Enter Project ID (optional, leave blank if unknown): ")
		projectID, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		projectID = strings.TrimSpace(projectID)

		cfg := &config.Config{
			BaseURL:   baseURL,
			ProjectID: projectID,
			Cookie:    cookie,
			Email:     email,
			Password:  password,
		}

		if err := config.Save(configPath, cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
		} else {
			fmt.Printf("Successfully initialized config in %s\n", configPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
