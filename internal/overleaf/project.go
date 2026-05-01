package overleaf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (c *Client) CreateProject(name string) (string, error) {
	domain := strings.Split(strings.Split(c.BaseURL, "://")[1], ":")[0]
	jsCode := fmt.Sprintf(`
async (page) => {
    const projectName = "%s";
    await page.goto('%s/project');
    
    await page.context().addCookies([{
        "name": "overleaf.sid",
        "value": "%s",
        "domain": "%s",
        "path": "/"
    }]);
    await page.reload();

    await page.getByRole('button', { name: 'New project' }).click();
    await page.getByRole('menuitem', { name: 'Blank project' }).click();
    await page.getByLabel('Project name').fill(projectName);
    await page.getByRole('button', { name: 'Create' }).click();
    
    await page.waitForURL(/\/project\/[a-f0-9]+/);
    return page.url().split('/').pop();
}
`, name, c.BaseURL, c.Cookie, domain)

	tempJS := filepath.Join(os.TempDir(), fmt.Sprintf("create_project_%s.js", uuid.New().String()))
	if err := os.WriteFile(tempJS, []byte(jsCode), 0644); err != nil {
		return "", err
	}
	defer os.Remove(tempJS)

	fmt.Printf("Creating project '%s' via Playwright...\n", name)
	cmd := exec.Command("npx", "playwright-cli", "run-code", fmt.Sprintf("--filename=%s", tempJS), "--raw")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("project creation failed: %v - %s", err, string(out))
	}

	pid := strings.TrimSpace(strings.Trim(string(out), "\""))
	if pid == "" {
		return "", fmt.Errorf("failed to get project ID from Playwright output")
	}

	c.ProjectID = pid
	_ = c.RefreshCSRF()
	return pid, nil
}

func (c *Client) DeleteProject(projectID string) error {
	pid := projectID
	if pid == "" {
		pid = c.ProjectID
	}
	if pid == "" {
		return fmt.Errorf("no project ID specified for deletion")
	}

	domain := strings.Split(strings.Split(c.BaseURL, "://")[1], ":")[0]
	jsCode := fmt.Sprintf(`
async (page) => {
    const pid = "%s";
    await page.goto('%s/project');
    
    await page.context().addCookies([{
        "name": "overleaf.sid",
        "value": "%s",
        "domain": "%s",
        "path": "/"
    }]);
    await page.reload();

    await page.evaluate((id) => {
        const btn = document.querySelector(` + "`" + `button[onclick*="${id}"][onclick*="trash"]` + "`" + `) || 
                    document.querySelector(` + "`" + `a[href*="${id}"]` + "`" + `).closest('tr').querySelector('button[aria-label*="Trash"]');
        if (btn) btn.click();
    }, pid);
    
    try {
        await page.getByRole('button', { name: 'Confirm' }).click();
    } catch (e) {
        // Confirmation might not be needed
    }
    
    await page.goto('%s/project/trash');
    await page.evaluate((id) => {
        const btn = document.querySelector(` + "`" + `button[onclick*="${id}"][onclick*="delete"]` + "`" + `) || 
                          document.querySelector(` + "`" + `a[href*="${id}"]` + "`" + `).closest('tr').querySelector('button[aria-label*="Delete"]');
        if (btn) btn.click();
    }, pid);
    await page.getByRole('button', { name: 'Confirm' }).click();
    return "OK";
}
`, pid, c.BaseURL, c.Cookie, domain, c.BaseURL)

	tempJS := filepath.Join(os.TempDir(), fmt.Sprintf("delete_project_%s.js", uuid.New().String()))
	if err := os.WriteFile(tempJS, []byte(jsCode), 0644); err != nil {
		return err
	}
	defer os.Remove(tempJS)

	fmt.Printf("Deleting project %s via Playwright...\n", pid)
	cmd := exec.Command("npx", "playwright-cli", "run-code", fmt.Sprintf("--filename=%s", tempJS), "--raw")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("project deletion failed: %v - %s", err, string(out))
	}

	return nil
}
