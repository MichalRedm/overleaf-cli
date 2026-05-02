package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"overleaf-cli/internal/config"
	"overleaf-cli/internal/overleaf"
	"overleaf-cli/internal/state"

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
		force, _ := cmd.Flags().GetBool("force")

		statePath := filepath.Join(config.MetadataDir, "state.json")
		projState, err := state.Load(statePath)
		if err != nil {
			fmt.Printf("Warning: could not load state: %v. Performing full sync.\n", err)
			projState = state.NewProjectState(statePath)
			force = true
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
		uploadedCount := 0
		skippedCount := 0

		err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if path == src {
				return nil
			}

			relPath, _ := filepath.Rel(src, path)
			relPath = filepath.ToSlash(relPath)

			// Skip hidden files/dirs except .overleaf (though we should skip it too as it's metadata)
			if strings.HasPrefix(filepath.Base(path), ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			localEntities[relPath] = true

			if !info.IsDir() {
				hash, err := state.CalculateHash(path)
				if err != nil {
					fmt.Printf("Error hashing %s: %v\n", relPath, err)
					return nil
				}

				if !force {
					if s, ok := projState.Files[relPath]; ok && s.Hash == hash {
						skippedCount++
						return nil
					}
				}

				if err := client.UploadFile(path, relPath, rootID, em); err != nil {
					fmt.Printf("Error uploading %s: %v\n", relPath, err)
				} else {
					projState.Files[relPath] = state.FileState{
						Hash: hash,
						Size: info.Size(),
					}
					uploadedCount++
				}
				// Add a small delay between uploads to avoid hitting rate limits
				time.Sleep(200 * time.Millisecond)
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking local directory: %v\n", err)
		}

		fmt.Printf("Push complete: %d uploaded, %d skipped\n", uploadedCount, skippedCount)

		if deleteRemote {
			fmt.Println("Pruning remote entities not present locally...")
			// Refresh entities to get latest state
			em, err = client.GetEntities()
			if err != nil {
				fmt.Printf("Error refreshing entities for pruning: %v\n", err)
			} else {
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

				deletedCount := 0
				for _, path := range toDelete {
					ent := em.Entities[path]
					if ent.Type == overleaf.EntityFolder {
						// Check if any local file is inside this folder
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
					} else {
						delete(projState.Files, path)
						deletedCount++
					}
				}
				fmt.Printf("Pruning complete: %d deleted\n", deletedCount)
			}
		}

		if err := projState.Save(); err != nil {
			fmt.Printf("Error saving state: %v\n", err)
		}
	},
}

func init() {
	pushCmd.Flags().StringP("src", "s", ".", "source directory")
	pushCmd.Flags().Bool("delete", false, "delete remote files not found locally")
	pushCmd.Flags().Bool("force", false, "force upload of all files")
	rootCmd.AddCommand(pushCmd)
}
