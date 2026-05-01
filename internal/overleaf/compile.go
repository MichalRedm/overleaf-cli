package overleaf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

func (c *Client) Compile() error {
	payload := map[string]interface{}{
		"check":                      "silent",
		"draft":                      false,
		"incrementalCompilesEnabled": true,
		"rootDoc_id":                 "",
		"stopOnFirstError":           false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	compileURL := fmt.Sprintf("%s/project/%s/compile?enable_pdf_caching=true", c.BaseURL, c.ProjectID)
	req, err := http.NewRequest("POST", compileURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", c.CSRF)
	req.Header.Set("Referer", fmt.Sprintf("%s/project/%s", c.BaseURL, c.ProjectID))

	fmt.Println("Compiling project...")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("compile failed: %d", resp.StatusCode)
	}

	var res struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	fmt.Printf("Compilation status: %s\n", res.Status)

	c.ShowLogs()

	if res.Status == "success" {
		fmt.Println("Compilation successful! Use 'pdf' command to download.")
	}

	return nil
}

func (c *Client) ShowLogs() {
	findCmd := fmt.Sprintf("find /var/lib/overleaf/data/compiles -name 'output.log' | grep %s | xargs ls -t | head -n 1", c.ProjectID)
	cmd := exec.Command("docker", "exec", "sharelatex", "sh", "-c", findCmd)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to find logs via Docker: %v\n", err)
		return
	}

	logPath := strings.TrimSpace(string(out))
	if logPath == "" {
		fmt.Println("Could not find log file in container.")
		return
	}

	fmt.Printf("Reading logs from container: %s\n", logPath)
	catCmd := exec.Command("docker", "exec", "sharelatex", "cat", logPath)
	logOut, err := catCmd.Output()
	if err != nil {
		fmt.Printf("Failed to read logs: %v\n", err)
		return
	}
	lines := strings.Split(string(logOut), "\n")

	fmt.Println("\n--- LaTeX Errors and Warnings ---")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "! ") {
			fmt.Printf("\n[ERROR] Line %d: %s\n", i+1, line)
			for j := 1; j < 10; j++ {
				if i+j < len(lines) {
					fmt.Printf("  %s\n", lines[i+j])
				}
			}
			found = true
		} else if strings.Contains(strings.ToLower(line), "warning:") {
			if !strings.Contains(strings.ToLower(line), "overfull") && !strings.Contains(strings.ToLower(line), "underfull") {
				fmt.Printf("[WARNING] %s\n", line)
				found = true
			}
		}
	}

	if !found {
		fmt.Println("No obvious errors or warnings found in the log.")
	}
	fmt.Println("--- End of Logs ---")
}

func (c *Client) DownloadPDF(outputPath string) error {
	findCmd := fmt.Sprintf("find /var/lib/overleaf/data/compiles -name 'output.pdf' | grep %s | xargs ls -t | head -n 1", c.ProjectID)
	cmd := exec.Command("docker", "exec", "sharelatex", "sh", "-c", findCmd)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find PDF via Docker: %v", err)
	}

	pdfPath := strings.TrimSpace(string(out))
	if pdfPath == "" {
		return fmt.Errorf("could not find PDF file in container. Did it compile successfully?")
	}

	fmt.Printf("Downloading PDF from container: %s to %s\n", pdfPath, outputPath)
	cpCmd := exec.Command("docker", "cp", fmt.Sprintf("sharelatex:%s", pdfPath), outputPath)
	if err := cpCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy PDF: %v", err)
	}

	fmt.Printf("Successfully saved PDF to %s\n", outputPath)
	return nil
}
