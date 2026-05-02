package overleaf

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	BaseURL    string
	ProjectID  string
	AuthType    string
	AuthCommand string
	Cookie      string
	CookieName  string
	UseDocker   bool
	HTTP       *http.Client
	CSRF       string
}
func NewClient(baseURL, projectID, cookie, authType, authCommand string, useDocker bool) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	cookieNames := []string{"overleaf.sid", "sharelatex.sid"}

	client := &Client{
		BaseURL:     strings.TrimSuffix(baseURL, "/"),
		ProjectID:   projectID,
		AuthType:    authType,
		AuthCommand: authCommand,
		UseDocker:   useDocker,
		Cookie:      cookie,
		HTTP: &http.Client{
			Jar: jar,
		},
	}
	
	// Add a standard User-Agent
	client.HTTP.Transport = &uaRoundTripper{
		rt: http.DefaultTransport,
		ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	if projectID != "" && cookie != "" {
		// Auto-detect cookie name
		success := false
		for _, name := range cookieNames {
			// Clear jar and set only this cookie
			newJar, _ := cookiejar.New(nil)
			newJar.SetCookies(u, []*http.Cookie{{Name: name, Value: cookie}})
			client.HTTP.Jar = newJar
			client.CookieName = name
			
			if err := client.RefreshCSRF(); err == nil {
				fmt.Printf("Auto-detected cookie name: %s\n", name)
				success = true
				break
			}
		}
		if !success {
			fmt.Printf("Warning: Failed to auto-detect cookie name or cookie is invalid.\n")
			// Default back to sharelatex.sid for university instances if detection fails
			client.CookieName = "sharelatex.sid"
		}
	} else if projectID != "" {
		if err := client.RefreshCSRF(); err != nil {
			fmt.Printf("Initial CSRF refresh failed: %v\n", err)
		}
	}

	return client, nil
}

func (c *Client) RefreshCSRF() error {
	projectURL := fmt.Sprintf("%s/project/%s", c.BaseURL, c.ProjectID)
	req, _ := http.NewRequest("GET", projectURL, nil)
	resp, err := c.DoWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to connect to Overleaf project: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	html := string(body)

	// Try regex for window.csrfToken
	re := regexp.MustCompile(`window\.csrfToken\s*=\s*"([^"]+)"`)
	match := re.FindStringSubmatch(html)
	if len(match) > 1 {
		c.CSRF = match[1]
		return nil
	}

	// Try goquery for meta tag
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
		csrf, exists := doc.Find("meta[name='ol-csrfToken']").Attr("content")
		if exists {
			c.CSRF = csrf
			return nil
		}
	}

	// Try hidden input
	reInput := regexp.MustCompile(`input name="_csrf" type="hidden" value="([^"]+)"`)
	matchInput := reInput.FindStringSubmatch(html)
	if len(matchInput) > 1 {
		c.CSRF = matchInput[1]
		return nil
	}

	return fmt.Errorf("CSRF token not found in project page")
}

func (c *Client) IsAuthenticated() bool {
	// Simple check: can we access the project page?
	projectURL := fmt.Sprintf("%s/project/%s", c.BaseURL, c.ProjectID)
	req, _ := http.NewRequest("GET", projectURL, nil)
	resp, err := c.DoWithRetry(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// If we were redirected to a login page, we are not authenticated
	if strings.Contains(resp.Request.URL.Path, "/login") {
		return false
	}
	
	return resp.StatusCode == 200
}

func (c *Client) Login(email, password string) error {
	switch c.AuthType {
	case "custom":
		return c.loginCustom(email, password)
	default:
		return c.loginStandard(email, password)
	}
}

func (c *Client) loginStandard(email, password string) error {
	fmt.Printf("Attempting standard login to %s as %s...\n", c.BaseURL, email)

	// 1. Get Login Page for initial CSRF
	loginURL := fmt.Sprintf("%s/login", c.BaseURL)
	resp, err := c.HTTP.Get(loginURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	csrf, exists := doc.Find("input[name='_csrf']").Attr("value")
	if !exists {
		// Try meta tag
		csrf, exists = doc.Find("meta[name='ol-csrfToken']").Attr("content")
	}

	if !exists {
		return fmt.Errorf("could not find CSRF token for login")
	}

	// 2. Post credentials
	data := url.Values{}
	data.Set("email", email)
	data.Set("password", password)
	data.Set("_csrf", csrf)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Avoid redirects to follow the cookie change
	c.HTTP.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() { c.HTTP.CheckRedirect = nil }()

	resp, err = c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 302 {
		return fmt.Errorf("login failed: expected 302 redirect, got %d", resp.StatusCode)
	}

	return c.captureSessionCookie()
}

func (c *Client) loginCustom(email, password string) error {
	if c.AuthCommand == "" {
		return fmt.Errorf("auth_type is 'custom' but auth_command is not specified")
	}

	fmt.Printf("Running custom authentication command: %s\n", c.AuthCommand)

	cmd := exec.Command("sh", "-c", c.AuthCommand)
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		cmd = exec.Command("cmd", "/C", c.AuthCommand)
	}

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OVERLEAF_EMAIL=%s", email),
		fmt.Sprintf("OVERLEAF_PASSWORD=%s", password),
		fmt.Sprintf("OVERLEAF_URL=%s", c.BaseURL),
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("custom auth command failed with status %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return fmt.Errorf("failed to run custom auth command: %v", err)
	}

	cookieValue := strings.TrimSpace(string(out))
	if cookieValue == "" {
		return fmt.Errorf("custom auth command returned empty cookie")
	}

	// Set the cookie in the jar for both common names
	u, _ := url.Parse(c.BaseURL)
	c.HTTP.Jar.SetCookies(u, []*http.Cookie{
		{Name: "overleaf.sid", Value: cookieValue},
		{Name: "sharelatex.sid", Value: cookieValue},
	})

	c.Cookie = cookieValue
	// Try to find which one works by refreshing CSRF
	return c.captureSessionCookie()
}

func (c *Client) captureSessionCookie() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	cookies := c.HTTP.Jar.Cookies(u)
	cookieNames := []string{"overleaf.sid", "sharelatex.sid"}
	
	for _, cookie := range cookies {
		for _, name := range cookieNames {
			if cookie.Name == name {
				c.Cookie = cookie.Value
				c.CookieName = name
				fmt.Printf("Successfully captured session cookie: %s\n", name)
				return c.RefreshCSRF()
			}
		}
	}

	return fmt.Errorf("login succeeded but session cookie (overleaf.sid/sharelatex.sid) not found")
}

type uaRoundTripper struct {
	rt http.RoundTripper
	ua string
}

func (t *uaRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.ua)
	return t.rt.RoundTrip(req)
}

func (c *Client) DoWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	maxRetries := 10
	backoff := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Clone request body for retries if it's not nil
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		resp, err = c.HTTP.Do(req)
		
		// Restore body for next attempt if needed
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 429 {
			fmt.Printf("Rate limited (429), retrying in %v... (attempt %d/%d)\n", backoff, i+1, maxRetries)
			resp.Body.Close()
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		return resp, nil
	}

	return resp, fmt.Errorf("max retries reached for 429 errors")
}
