package overleaf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (c *Client) CreateProject(name string) (string, error) {
	domain := strings.Split(strings.Split(c.BaseURL, "://")[1], ":")[0]
	jsCode := fmt.Sprintf(`
async (page) => {
    // Wait for browser session to stabilize
    await page.waitForTimeout(2000);
    const projectName = "%s";
    
    // Set cookie before navigation to avoid redirecting to login page
    await page.context().addCookies([{
        "name": "%s",
        "value": "%s",
        "domain": "%s",
        "path": "/"
    }]);
    
    // Attempt navigation with a retry in case of transient errors
    try {
        await page.goto('%s/project');
    } catch (e) {
        await page.waitForTimeout(2000);
        await page.goto('%s/project');
    }

    // Select the "New project" button or link (supporting both English and Polish)
    const newProj = page.locator('.btn-new-project, a:has-text("New Project"), button:has-text("New Project"), a:has-text("Nowy projekt"), button:has-text("Nowy projekt")').first();
    await newProj.click();

    // Select the "Blank project" menu item or link
    const blankProj = page.locator('a:has-text("Blank Project"), [role="menuitem"]:has-text("Blank Project"), a:has-text("Pusty projekt"), [role="menuitem"]:has-text("Pusty projekt")').first();
    await blankProj.click();

    // Fill project name
    const nameInput = page.locator('input[name="name"], input[placeholder*="Project Name"], input[placeholder*="Nazwa projektu"]').first();
    await nameInput.fill(projectName);

    // Click "Create" button
    const createBtn = page.locator('button[type="submit"]:has-text("Create"), button:has-text("Create"), button[type="submit"]:has-text("Utwórz"), button:has-text("Utwórz")').first();
    await createBtn.click();
    
    await page.waitForURL(/\/project\/[a-f0-9]+/);
    return page.url().split('/').pop();
}
`, name, c.CookieName, c.Cookie, domain, c.BaseURL, c.BaseURL)

	tempJS := filepath.Join(".", fmt.Sprintf("create_project_%s.js", uuid.New().String()))
	if err := os.WriteFile(tempJS, []byte(jsCode), 0644); err != nil {
		return "", err
	}
	defer os.Remove(tempJS)

	// Start browser session
	fmt.Println("Opening temporary Playwright session...")
	openCmd := exec.Command("npx", "playwright-cli", "open")
	if err := openCmd.Start(); err != nil {
		return "", fmt.Errorf("failed to open playwright-cli session: %v", err)
	}
	// Give it a moment to initialize
	time.Sleep(3 * time.Second)
	defer func() {
		fmt.Println("Closing temporary Playwright session...")
		_ = exec.Command("npx", "playwright-cli", "close").Run()
	}()

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
    // Wait for browser session to stabilize
    await page.waitForTimeout(2000);
    const pid = "%s";
    
    // Set cookie before navigation to avoid redirecting to login page
    await page.context().addCookies([{
        "name": "%s",
        "value": "%s",
        "domain": "%s",
        "path": "/"
    }]);
    
    // Attempt navigation with a retry in case of transient errors
    try {
        await page.goto('%s/project');
    } catch (e) {
        await page.waitForTimeout(2000);
        await page.goto('%s/project');
    }

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
`, pid, c.CookieName, c.Cookie, domain, c.BaseURL, c.BaseURL, c.BaseURL)

	tempJS := filepath.Join(".", fmt.Sprintf("delete_project_%s.js", uuid.New().String()))
	if err := os.WriteFile(tempJS, []byte(jsCode), 0644); err != nil {
		return err
	}
	defer os.Remove(tempJS)

	// Start browser session
	fmt.Println("Opening temporary Playwright session...")
	openCmd := exec.Command("npx", "playwright-cli", "open")
	if err := openCmd.Start(); err != nil {
		return fmt.Errorf("failed to open playwright-cli session: %v", err)
	}
	// Give it a moment to initialize
	time.Sleep(3 * time.Second)
	defer func() {
		fmt.Println("Closing temporary Playwright session...")
		_ = exec.Command("npx", "playwright-cli", "close").Run()
	}()

	fmt.Printf("Deleting project %s via Playwright...\n", pid)
	cmd := exec.Command("npx", "playwright-cli", "run-code", fmt.Sprintf("--filename=%s", tempJS), "--raw")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("project deletion failed: %v - %s", err, string(out))
	}

	return nil
}
