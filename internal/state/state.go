package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

type FileState struct {
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	RemoteID string `json:"remote_id,omitempty"`
}

type ProjectState struct {
	Files map[string]FileState `json:"files"`
	path  string
}

func NewProjectState(path string) *ProjectState {
	return &ProjectState{
		Files: make(map[string]FileState),
		path:  path,
	}
}

func Load(path string) (*ProjectState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewProjectState(path), nil
		}
		return nil, err
	}

	state := NewProjectState(path)
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *ProjectState) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func CalculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
