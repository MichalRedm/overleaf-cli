package overleaf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (c *Client) GetOrCreateFolder(path string, rootID string, em *EntityMap) (string, error) {
	path = strings.Trim(strings.ReplaceAll(path, "\\", "/"), "/")
	if path == "" {
		return rootID, nil
	}

	if id, ok := em.Folders[path]; ok {
		return id, nil
	}

	parts := strings.Split(path, "/")
	currentID := rootID
	currentPath := ""

	for _, part := range parts {
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		if id, ok := em.Folders[currentPath]; ok {
			currentID = id
			continue
		}

		fmt.Printf("Creating folder %s in %s...\n", part, currentID)
		payload := map[string]string{
			"name":             part,
			"parent_folder_id": currentID,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/project/%s/folder", c.BaseURL, c.ProjectID), bytes.NewBuffer(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Csrf-Token", c.CSRF)
		req.Header.Set("Referer", fmt.Sprintf("%s/project/%s", c.BaseURL, c.ProjectID))

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			var res struct {
				ID string `json:"id"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				return "", err
			}
			em.Folders[currentPath] = res.ID
			currentID = res.ID
		} else {
			// Check if already exists
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				respBody = []byte("could not read error body")
			}
			if strings.Contains(string(respBody), "already exists") {
				// Refresh entities and try again
				newEm, err := c.GetEntities()
				if err != nil {
					return "", err
				}
				*em = *newEm
				if id, ok := em.Folders[currentPath]; ok {
					currentID = id
					continue
				}
			}
			return "", fmt.Errorf("failed to create folder %s: %d - %s", part, resp.StatusCode, string(respBody))
		}
	}

	return currentID, nil
}

func (c *Client) UploadFile(localPath string, remotePath string, rootID string, em *EntityMap) error {
	relDir := filepath.ToSlash(filepath.Dir(remotePath))
	folderID, err := c.GetOrCreateFolder(relDir, rootID, em)
	if err != nil {
		return err
	}

	filename := filepath.Base(remotePath)
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, _ := file.Stat()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add CSRF
	if err := writer.WriteField("_csrf", c.CSRF); err != nil {
		return err
	}
	if err := writer.WriteField("qquuid", uuid.New().String()); err != nil {
		return err
	}
	if err := writer.WriteField("qqfilename", filename); err != nil {
		return err
	}
	if err := writer.WriteField("name", filename); err != nil {
		return err
	}
	if err := writer.WriteField("qqtotalfilesize", fmt.Sprintf("%d", fileInfo.Size())); err != nil {
		return err
	}

	part, err := writer.CreateFormFile("qqfile", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	uploadURL := fmt.Sprintf("%s/project/%s/upload?folder_id=%s", c.BaseURL, c.ProjectID, folderID)
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Csrf-Token", c.CSRF)
	req.Header.Set("Referer", fmt.Sprintf("%s/project/%s", c.BaseURL, c.ProjectID))

	fmt.Printf("Uploading %s to folder %s...\n", filename, folderID)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			fmt.Println("CSRF might have expired, refreshing...")
			_ = c.RefreshCSRF()
		}
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			respBody = []byte("could not read error body")
		}
		return fmt.Errorf("failed to upload %s: %d - %s", filename, resp.StatusCode, string(respBody))
	}

	var res struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if !res.Success {
		return fmt.Errorf("server rejected %s: %s", filename, res.Error)
	}

	fmt.Printf("Successfully uploaded %s\n", filename)
	return nil
}

func (c *Client) DeleteEntity(entityID string, entityType EntityType) error {
	url := fmt.Sprintf("%s/project/%s/%s/%s", c.BaseURL, c.ProjectID, entityType, entityID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Csrf-Token", c.CSRF)
	req.Header.Set("Referer", fmt.Sprintf("%s/project/%s", c.BaseURL, c.ProjectID))

	fmt.Printf("Deleting %s %s...\n", entityType, entityID)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			respBody = []byte("could not read error body")
		}
		return fmt.Errorf("failed to delete %s %s: %d - %s", entityType, entityID, resp.StatusCode, string(respBody))
	}

	return nil
}
