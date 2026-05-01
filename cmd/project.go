package cmd

import (
	"fmt"
	"overleaf-cli/internal/config"
	"overleaf-cli/internal/overleaf"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects on Overleaf",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		name, _ := cmd.Flags().GetString("name")
		
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		client, err := overleaf.NewClient(cfg.BaseURL, cfg.ProjectID, cfg.Cookie)
		if err != nil {
			fmt.Printf("Error creating client: %v\n", err)
			return
		}

		newID, err := client.CreateProject(name)
		if err != nil {
			fmt.Printf("Error creating project: %v\n", err)
			return
		}

		fmt.Printf("Successfully created project '%s' with ID: %s\n", name, newID)
		cfg.ProjectID = newID
		if err := config.Save(configPath, cfg); err != nil {
			fmt.Printf("Error saving updated config: %v\n", err)
		} else {
			fmt.Printf("Updated %s with new project_id\n", configPath)
		}
	},
}

var projectRmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Delete a project",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		id, _ := cmd.Flags().GetString("id")
		
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		client, err := overleaf.NewClient(cfg.BaseURL, cfg.ProjectID, cfg.Cookie)
		if err != nil {
			fmt.Printf("Error creating client: %v\n", err)
			return
		}

		if id == "" {
			id = cfg.ProjectID
		}

		if err := client.DeleteProject(id); err != nil {
			fmt.Printf("Error deleting project: %v\n", err)
		} else {
			fmt.Printf("Successfully deleted project %s\n", id)
		}
	},
}

func init() {
	projectCreateCmd.Flags().StringP("name", "n", "", "project name")
	projectCreateCmd.MarkFlagRequired("name")
	
	projectRmCmd.Flags().String("id", "", "project ID to delete (defaults to config ID)")
	
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectRmCmd)
	rootCmd.AddCommand(projectCmd)
}
