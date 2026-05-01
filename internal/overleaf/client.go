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
	jar, _ := cookiejar.New(nil)
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
