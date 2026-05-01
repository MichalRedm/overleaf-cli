package overleaf

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Client struct {
	BaseURL   string
	ProjectID string
	Cookie    string
	HTTP      *http.Client
	CSRF      string
}

func NewClient(baseURL, projectID, cookie string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "overleaf.sid",
			Value: cookie,
		},
	})

	client := &Client{
		BaseURL:   strings.TrimSuffix(baseURL, "/"),
		ProjectID: projectID,
		Cookie:    cookie,
		HTTP: &http.Client{
			Jar: jar,
		},
	}

	if projectID != "" {
		if err := client.RefreshCSRF(); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *Client) RefreshCSRF() error {
	projectURL := fmt.Sprintf("%s/project/%s", c.BaseURL, c.ProjectID)
	resp, err := c.HTTP.Get(projectURL)
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
	resp, err := c.HTTP.Get(projectURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *Client) Login(email, password string) error {
	fmt.Printf("Attempting login to %s as %s...\n", c.BaseURL, email)

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

	// Capture new cookie
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	cookies := c.HTTP.Jar.Cookies(u)
	for _, cookie := range cookies {
		if cookie.Name == "overleaf.sid" {
			c.Cookie = cookie.Value
			fmt.Println("Successfully logged in and updated session cookie.")
			return c.RefreshCSRF()
		}
	}

	return fmt.Errorf("login succeeded but overleaf.sid cookie not found")
}
