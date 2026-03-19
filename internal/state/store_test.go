package state

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := &State{LastAuthor: "fponin"}
	s.AddCommitMessage("first commit")
	s.AddService("discovery")

	if err := Save(path, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.LastAuthor != "fponin" {
		t.Errorf("unexpected author: %s", loaded.LastAuthor)
	}
	if len(loaded.RecentCommitMessages) != 1 || loaded.RecentCommitMessages[0] != "first commit" {
		t.Errorf("unexpected commit messages: %v", loaded.RecentCommitMessages)
	}
}

func TestLoadMissingFile(t *testing.T) {
	s, err := Load("/nonexistent/state.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil state")
	}
}

func TestAddCommitMessageFIFO(t *testing.T) {
	s := &State{}
	for i := 0; i < 15; i++ {
		s.AddCommitMessage("msg")
	}
	// After dedup, should still be only 1 unique entry
	if len(s.RecentCommitMessages) != 1 {
		t.Errorf("expected 1 entry after dedup, got %d", len(s.RecentCommitMessages))
	}

	// Add unique messages
	s2 := &State{}
	for i := 0; i < 15; i++ {
		s2.AddCommitMessage(filepath.Join("msg", string(rune('a'+i))))
	}
	if len(s2.RecentCommitMessages) != maxCommitMessages {
		t.Errorf("expected %d entries, got %d", maxCommitMessages, len(s2.RecentCommitMessages))
	}
}

func TestAddServiceDedup(t *testing.T) {
	s := &State{}
	s.AddService("discovery")
	s.AddService("cart")
	s.AddService("discovery") // duplicate

	if len(s.RecentServices) != 2 {
		t.Errorf("expected 2 services after dedup, got %d", len(s.RecentServices))
	}
	if s.RecentServices[0] != "discovery" {
		t.Error("discovery should be first after re-add")
	}
}

func TestAddOutputFileCap(t *testing.T) {
	s := &State{}
	for i := 0; i < 10; i++ {
		s.AddOutputFile(filepath.Join("file", string(rune('a'+i))))
	}
	if len(s.RecentOutputFiles) != maxOutputFiles {
		t.Errorf("expected %d output files, got %d", maxOutputFiles, len(s.RecentOutputFiles))
	}
}
