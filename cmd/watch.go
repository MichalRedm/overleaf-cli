package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"overleaf-cli/internal/overleaf"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for local changes and sync automatically",
	Run: func(cmd *cobra.Command, args []string) {
		client, cfg := getClient(cmd)
		if client == nil {
			return
		}

		src := resolveSrcPath(cmd)
		deleteRemote, err := cmd.Flags().GetBool("delete")
		if err != nil {
			deleteRemote = false
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		done := make(chan bool)
		
		// Helper to re-add directories to watch
		addDirs := func(root string) {
			if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err == nil && info.IsDir() {
					if !strings.HasPrefix(filepath.Base(path), ".") {
						if err := watcher.Add(path); err != nil {
							fmt.Printf("Warning: failed to watch directory %s: %v\n", path, err)
						}
					}
				}
				return nil
			}); err != nil {
				fmt.Printf("Warning: error walking directory %s: %v\n", root, err)
			}
		}

		addDirs(src)

		fmt.Printf("Watching for changes in %s...\n", src)

		// Simple debouncing
		var timer *time.Timer
		debounce := 500 * time.Millisecond

		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					
					// Ignore hidden files
					if strings.HasPrefix(filepath.Base(event.Name), ".") {
						continue
					}

					fmt.Printf("Event: %s\n", event)
					
					if event.Op&fsnotify.Create == fsnotify.Create {
						info, err := os.Stat(event.Name)
						if err == nil && info.IsDir() {
							if err := watcher.Add(event.Name); err != nil {
								fmt.Printf("Warning: failed to watch new directory %s: %v\n", event.Name, err)
							}
						}
					}

					if timer != nil {
						timer.Stop()
					}
					
					timer = time.AfterFunc(debounce, func() {
						fmt.Println("Changes detected, syncing...")
						// We just call the push logic here
						// For simplicity in this demo, we re-run the push logic
						// In a real tool, we might want to only sync the changed file
						push(client, src, deleteRemote, cfg.RootFolderID)
					})

				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("error:", err)
				}
			}
		}()

		<-done
	},
}

// push is a helper that contains the push logic (refactored from push.go)
func push(client *overleaf.Client, src string, deleteRemote bool, configRootID string) {
	em, err := client.GetEntities()
	if err != nil {
		fmt.Printf("Error retrieving entities: %v\n", err)
		return
	}

	rootID := em.RootID
	if rootID == "" {
		rootID = configRootID
	}

	localEntities := make(map[string]bool)
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		relPath, _ := filepath.Rel(src, path)
		relPath = filepath.ToSlash(relPath)
		if strings.HasPrefix(filepath.Base(path), ".") {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		localEntities[relPath] = true
		if !info.IsDir() {
			if err := client.UploadFile(path, relPath, rootID, em); err != nil {
				fmt.Printf("Error uploading %s: %v\n", relPath, err)
			}
		}
		return nil
	}); err != nil {
		fmt.Printf("Error walking local directory for push: %v\n", err)
	}

	if deleteRemote {
		fmt.Println("Warning: pruning is not yet implemented in watch mode.")
	}
}

func init() {
	watchCmd.Flags().StringP("src", "s", ".", "source directory")
	watchCmd.Flags().Bool("delete", false, "delete remote files not found locally")
	rootCmd.AddCommand(watchCmd)
}
