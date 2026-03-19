package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultStatePath returns the default state file path.
func DefaultStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hpub", "state.json")
}

// Load reads state from the given path. Returns a fresh State if the file doesn't exist.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &State{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	return &s, nil
}

// Save writes the state to the given path, creating parent directories as needed.
func Save(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing state: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// AddCommitMessage prepends msg to recent commit messages, keeping at most maxCommitMessages.
func (s *State) AddCommitMessage(msg string) {
	if msg == "" {
		return
	}
	s.RecentCommitMessages = prependDedup(s.RecentCommitMessages, msg, maxCommitMessages)
}

// AddService prepends service to recent services, keeping at most maxServices.
func (s *State) AddService(service string) {
	if service == "" {
		return
	}
	s.RecentServices = prependDedup(s.RecentServices, service, maxServices)
}

// AddOutputFile prepends file to recent output files, keeping at most maxOutputFiles.
func (s *State) AddOutputFile(file string) {
	if file == "" {
		return
	}
	s.RecentOutputFiles = prependDedup(s.RecentOutputFiles, file, maxOutputFiles)
}

// prependDedup adds item to the front, removes duplicates, and caps the slice at max.
func prependDedup(slice []string, item string, max int) []string {
	// Remove existing occurrence
	filtered := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != item {
			filtered = append(filtered, v)
		}
	}
	// Prepend
	result := append([]string{item}, filtered...)
	if len(result) > max {
		result = result[:max]
	}
	return result
}
