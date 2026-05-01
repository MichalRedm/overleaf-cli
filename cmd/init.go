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
		configPath, _ := cmd.Flags().GetString("config")
		
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter Overleaf Base URL (e.g., http://localhost:80): ")
		baseURL, _ := reader.ReadString('\n')
		baseURL = strings.TrimSpace(baseURL)

		fmt.Print("Enter overleaf.sid cookie value (optional, press enter to skip and use email/pass): ")
		cookie, _ := reader.ReadString('\n')
		cookie = strings.TrimSpace(cookie)

		fmt.Print("Enter Email: ")
		email, _ := reader.ReadString('\n')
		email = strings.TrimSpace(email)

		fmt.Print("Enter Password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		fmt.Print("Enter Project ID (optional, leave blank if unknown): ")
		projectID, _ := reader.ReadString('\n')
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
