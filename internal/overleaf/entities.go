package overleaf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type EntityType string

const (
	EntityFolder EntityType = "folder"
	EntityDoc    EntityType = "doc"
	EntityFile   EntityType = "file"
)

type RemoteEntity struct {
	ID   string     `json:"id"`
	Type EntityType `json:"type"`
}

type EntityMap struct {
	Folders  map[string]string       // path -> id
	Entities map[string]RemoteEntity // path -> info
	RootID   string
}

func (c *Client) GetEntities() (*EntityMap, error) {
	em := &EntityMap{
		Folders:  make(map[string]string),
		Entities: make(map[string]RemoteEntity),
	}

	// 1. Try Internal Websocket Discovery (native & most reliable)
	fmt.Println("Attempting native entity discovery via websocket...")
	if discovery, err := c.DiscoverEntitiesInternal(); err == nil {
		em.RootID = discovery.RootID
		em.Entities = discovery.Entities
		for path, info := range em.Entities {
			if info.Type == EntityFolder {
				em.Folders[path] = info.ID
			}
		}
		fmt.Println("Successfully discovered entities via native websocket")
		return em, nil
	} else {
		fmt.Printf("Native discovery failed: %v\n", err)
	}

	// 2. Try Docker/MongoDB (useful for local self-hosted)
	cmdStr := fmt.Sprintf("JSON.stringify(db.projects.findOne({_id: ObjectId('%s')}, {rootFolder: 1}).rootFolder)", c.ProjectID)
	cmd := exec.Command("docker", "exec", "mongo", "mongosh", "sharelatex", "--quiet", "--eval", cmdStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		var rootFolder map[string]interface{}
		output := out.String()
		// Clean output if needed (mongosh sometimes adds noise)
		start := strings.Index(output, "{")
		if start != -1 {
			if err := json.Unmarshal([]byte(output[start:]), &rootFolder); err == nil {
				c.parseRecursiveFolder(rootFolder, "", em)
				return em, nil
			}
		}
	}

	// Fallback to Web API
	apiURL := fmt.Sprintf("%s/project/%s/entities", c.BaseURL, c.ProjectID)
	resp, err := c.HTTP.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var data struct {
			Entities []map[string]interface{} `json:"entities"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
			c.parseFlatEntities(data.Entities, em)
			return em, nil
		}
	}

	return nil, fmt.Errorf("failed to retrieve entities from both Docker and Web API")
}

func (c *Client) parseRecursiveFolder(folder map[string]interface{}, path string, em *EntityMap) {
	id, _ := folder["_id"].(string)
	if id == "" {
		id, _ = folder["id"].(string)
	}
	name, _ := folder["name"].(string)

	var fullPath string
	if name == "rootFolder" {
		fullPath = ""
		em.RootID = id
	} else {
		if path == "" {
			fullPath = name
		} else {
			fullPath = path + "/" + name
		}
	}

	if fullPath != "" {
		em.Folders[fullPath] = id
		em.Entities[fullPath] = RemoteEntity{ID: id, Type: EntityFolder}
	} else {
		em.Folders[""] = id
	}

	if docs, ok := folder["docs"].([]interface{}); ok {
		for _, d := range docs {
			doc := d.(map[string]interface{})
			docID, _ := doc["_id"].(string)
			if docID == "" {
				docID, _ = doc["id"].(string)
			}
			docName, _ := doc["name"].(string)
			docPath := docName
			if fullPath != "" {
				docPath = fullPath + "/" + docName
			}
			em.Entities[docPath] = RemoteEntity{ID: docID, Type: EntityDoc}
		}
	}

	if files, ok := folder["fileRefs"].([]interface{}); ok {
		for _, f := range files {
			file := f.(map[string]interface{})
			fileID, _ := file["_id"].(string)
			if fileID == "" {
				fileID, _ = file["id"].(string)
			}
			fileName, _ := file["name"].(string)
			filePath := fileName
			if fullPath != "" {
				filePath = fullPath + "/" + fileName
			}
			em.Entities[filePath] = RemoteEntity{ID: fileID, Type: EntityFile}
		}
	}

	if folders, ok := folder["folders"].([]interface{}); ok {
		for _, f := range folders {
			c.parseRecursiveFolder(f.(map[string]interface{}), fullPath, em)
		}
	}
}

func (c *Client) parseFlatEntities(entities []map[string]interface{}, em *EntityMap) {
	// Simple implementation: map by ID to resolve paths
	idToEntity := make(map[string]map[string]interface{})
	for _, e := range entities {
		id, _ := e["_id"].(string)
		if id == "" {
			id, _ = e["id"].(string)
		}
		idToEntity[id] = e
	}

	var getPath func(string) string
	getPath = func(id string) string {
		e := idToEntity[id]
		parentID, _ := e["parentId"].(string)
		if parentID == "" {
			parentID, _ = e["parent_folder_id"].(string)
		}
		if parentID == "" {
			return ""
		}
		parentPath := getPath(parentID)
		name, _ := e["name"].(string)
		if parentPath == "" {
			return name
		}
		return parentPath + "/" + name
	}

	for id, e := range idToEntity {
		path := getPath(id)
		eTypeStr, _ := e["type"].(string)
		eType := EntityType(eTypeStr)

		if eType == EntityFolder {
			em.Folders[path] = id
			parentID, _ := e["parentId"].(string)
			if parentID == "" {
				parentID, _ = e["parent_folder_id"].(string)
			}
			if parentID == "" {
				em.RootID = id
			}
		}
		if path != "" {
			em.Entities[path] = RemoteEntity{ID: id, Type: eType}
		}
	}
}

func (c *Client) DiscoverEntitiesInternal() (*EntityMap, error) {
	// 1. Handshake
	handshakeURL := fmt.Sprintf("%s/socket.io/1/?projectId=%s&t=%d", c.BaseURL, c.ProjectID, time.Now().UnixMilli())
	req, _ := http.NewRequest("GET", handshakeURL, nil)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", c.CookieName, c.Cookie))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("handshake failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	parts := strings.Split(string(body), ":")
	if len(parts) < 1 || parts[0] == "" {
		return nil, fmt.Errorf("invalid handshake response: %s", string(body))
	}
	sessionID := parts[0]

	// 2. Connect via Websocket
	wsURL := fmt.Sprintf("%s/socket.io/1/websocket/%s?projectId=%s", strings.Replace(c.BaseURL, "http", "ws", 1), sessionID, c.ProjectID)
	config, err := websocket.NewConfig(wsURL, c.BaseURL+"/")
	if err != nil {
		return nil, fmt.Errorf("ws config failed: %w", err)
	}
	// Important: Cookie must be name=value
	config.Header.Set("Cookie", fmt.Sprintf("%s=%s", c.CookieName, c.Cookie))
	config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	ws, err := websocket.DialConfig(config)
	if err != nil {
		return nil, fmt.Errorf("ws dial failed: %w", err)
	}
	defer ws.Close()

	// 3. State machine: Send joinProject -> Wait for response
	em := &EntityMap{
		Folders:  make(map[string]string),
		Entities: make(map[string]RemoteEntity),
	}
	var rootFolder map[string]interface{}
	deadline := time.Now().Add(20 * time.Second)
	
	// Send joinProject immediately with ID 1
	if err := websocket.Message.Send(ws, fmt.Sprintf(`5:1::{"name":"joinProject","args":[{"project_id":"%s"}]}`, c.ProjectID)); err != nil {
		return nil, fmt.Errorf("failed to send joinProject: %w", err)
	}
	// Also send without ID just in case
	_ = websocket.Message.Send(ws, fmt.Sprintf(`5:::{"name":"joinProject","args":[{"project_id":"%s"}]}`, c.ProjectID))

	for time.Now().Before(deadline) {
		var reply string
		if err := websocket.Message.Receive(ws, &reply); err != nil {
			return nil, fmt.Errorf("ws receive failed: %w", err)
		}

		if strings.Contains(reply, "project:joined") || strings.Contains(reply, "joinProjectResponse") {
			// Extract the JSON part after 5:ID::
			start := strings.Index(reply, "{")
			if start == -1 {
				continue
			}
			var msg struct {
				Name string `json:"name"`
				Args []struct {
					Project struct {
						RootFolder interface{} `json:"rootFolder"`
					} `json:"project"`
				} `json:"args"`
			}
			if err := json.Unmarshal([]byte(reply[start:]), &msg); err != nil {
				continue
			}
			if len(msg.Args) > 0 {
				rf := msg.Args[0].Project.RootFolder
				// rootFolder can be an array or a single object
				if slice, ok := rf.([]interface{}); ok && len(slice) > 0 {
					rootFolder = slice[0].(map[string]interface{})
				} else if obj, ok := rf.(map[string]interface{}); ok {
					rootFolder = obj
				}
				
				if rootFolder != nil {
					break
				}
			}
		}
		
		// Handle heartbeats to keep connection alive
		if reply == "2::" {
			_ = websocket.Message.Send(ws, "2::")
		}
	}

	if rootFolder == nil {
		return nil, fmt.Errorf("timeout waiting for project tree")
	}

	c.parseRecursiveFolder(rootFolder, "", em)
	return em, nil
}
