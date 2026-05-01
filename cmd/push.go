package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"overleaf-cli/internal/overleaf"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Sync local files to Overleaf",
	Run: func(cmd *cobra.Command, args []string) {
		client, cfg := getClient(cmd)
		if client == nil {
			return
		}

		src, err := cmd.Flags().GetString("src")
		if err != nil {
			src = "."
		}
		deleteRemote, err := cmd.Flags().GetBool("delete")
		if err != nil {
			deleteRemote = false
		}

		em, err := client.GetEntities()
		if err != nil {
			fmt.Printf("Error retrieving entities: %v\n", err)
			return
		}

		rootID := em.RootID
		if rootID == "" {
			rootID = cfg.RootFolderID
		}
		if rootID == "" {
			fmt.Println("Error: Could not determine root folder ID.")
			return
		}

		fmt.Printf("Starting push from %s to root folder %s\n", src, rootID)

		localEntities := make(map[string]bool)
		err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if path == src {
				return nil
			}

			relPath, _ := filepath.Rel(src, path)
			relPath = filepath.ToSlash(relPath)

			// Skip hidden files/dirs
			if strings.HasPrefix(filepath.Base(path), ".") {
				if info, _ := os.Stat(path); info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			localEntities[relPath] = true

			if !info.IsDir() {
				if ent, ok := em.Entities[relPath]; ok {
					if ent.Type != overleaf.EntityFolder {
						if err := client.DeleteEntity(ent.ID, ent.Type); err != nil {
							fmt.Printf("Warning: failed to delete existing entity %s: %v\n", relPath, err)
						}
					}
				}
				if err := client.UploadFile(path, relPath, rootID, em); err != nil {
					fmt.Printf("Error uploading %s: %v\n", relPath, err)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking local directory: %v\n", err)
			return
		}

		if deleteRemote {
			fmt.Println("Pruning remote entities not present locally...")
			// Refresh entities to get latest state
			em, err = client.GetEntities()
			if err != nil {
				fmt.Printf("Error refreshing entities for pruning: %v\n", err)
				return
			}
			
			// Collect paths to delete (longest first to handle nested folders)
			var toDelete []string
			for path := range em.Entities {
				if !localEntities[path] {
					toDelete = append(toDelete, path)
				}
			}
			
			// Sort by length descending
			for i := 0; i < len(toDelete); i++ {
				for j := i + 1; j < len(toDelete); j++ {
					if len(toDelete[i]) < len(toDelete[j]) {
						toDelete[i], toDelete[j] = toDelete[j], toDelete[i]
					}
				}
			}

			for _, path := range toDelete {
				ent := em.Entities[path]
				if ent.Type == overleaf.EntityFolder {
					// Check if any local file is inside this folder (shouldn't happen if localEntities is correct)
					skip := false
					for lp := range localEntities {
						if strings.HasPrefix(lp, path+"/") {
							skip = true
							break
						}
					}
					if skip {
						continue
					}
				}
				fmt.Printf("Pruning %s (%s)...\n", path, ent.Type)
				if err := client.DeleteEntity(ent.ID, ent.Type); err != nil {
					fmt.Printf("Error pruning %s: %v\n", path, err)
				}
			}
		}
	},
}

func init() {
	pushCmd.Flags().StringP("src", "s", ".", "source directory")
	pushCmd.Flags().Bool("delete", false, "delete remote files not found locally")
	rootCmd.AddCommand(pushCmd)
}
