package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Add overleaf-cli to the system PATH",
	Long:  "Adds the directory containing the overleaf-cli binary to the user's system PATH environment variable.",
	Run: func(cmd *cobra.Command, args []string) {
		exePath, err := os.Executable()
		if err != nil {
			fmt.Printf("Error getting executable path: %v\n", err)
			return
		}

		exeDir := filepath.Dir(exePath)
		absDir, err := filepath.Abs(exeDir)
		if err != nil {
			fmt.Printf("Error getting absolute path: %v\n", err)
			return
		}

		fmt.Printf("Current binary directory: %s\n", absDir)

		switch runtime.GOOS {
		case "windows":
			installWindows(absDir)
		case "linux", "darwin":
			installUnix(absDir)
		default:
			fmt.Printf("Automatic PATH update not supported on %s. Please add %s to your PATH manually.\n", runtime.GOOS, absDir)
		}
	},
}

func installWindows(dir string) {
	// Use PowerShell to update the User PATH
	// We append the directory if it's not already there
	psCommand := fmt.Sprintf(`
		$oldPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
		if ($oldPath -split ";" -notcontains "%s") {
			$newPath = "$oldPath;%s"
			[System.Environment]::SetEnvironmentVariable("Path", $newPath, "User")
			Write-Output "Successfully added to User PATH."
		} else {
			Write-Output "Directory is already in PATH."
		}
	`, dir, dir)

	cmd := exec.Command("powershell", "-Command", psCommand)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error updating PATH: %v\nOutput: %s\n", err, string(out))
		return
	}
	fmt.Println(strings.TrimSpace(string(out)))
	fmt.Println("Note: You may need to restart your terminal for changes to take effect.")
}

func installUnix(dir string) {
	shell := os.Getenv("SHELL")
	var rcFile string

	if strings.Contains(shell, "zsh") {
		rcFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
	} else {
		rcFile = filepath.Join(os.Getenv("HOME"), ".bashrc")
	}

	exportCmd := fmt.Sprintf("\nexport PATH=\"$PATH:%s\"\n", dir)
	
	// Check if already in file
	content, _ := os.ReadFile(rcFile)
	if strings.Contains(string(content), dir) {
		fmt.Printf("Path already exists in %s\n", rcFile)
		return
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Error opening %s: %v\n", rcFile, err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(exportCmd); err != nil {
		fmt.Printf("Error writing to %s: %v\n", rcFile, err)
		return
	}

	fmt.Printf("Successfully added PATH export to %s\n", rcFile)
	fmt.Println("Please run 'source " + rcFile + "' or restart your terminal.")
}

func init() {
	rootCmd.AddCommand(installCmd)
}
