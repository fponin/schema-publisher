package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	def := DefaultConfig()
	if err := Save(path, &def); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Subgraphs) != len(def.Subgraphs) {
		t.Errorf("subgraph count: got %d, want %d", len(loaded.Subgraphs), len(def.Subgraphs))
	}

	if _, ok := loaded.Environments["dev"]; !ok {
		t.Error("dev environment missing after round-trip")
	}
	if _, ok := loaded.Environments["prod"]; !ok {
		t.Error("prod environment missing after round-trip")
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadOrDefault_NoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.yaml")
	cfg, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Subgraphs) == 0 {
		t.Error("expected default subgraphs")
	}
}

func TestExpandTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := ExpandTilde("~/foo/bar")
	want := filepath.Join(home, "foo", "bar")
	if got != want {
		t.Errorf("ExpandTilde: got %q, want %q", got, want)
	}

	got2 := ExpandTilde("/absolute/path")
	if got2 != "/absolute/path" {
		t.Errorf("ExpandTilde absolute: got %q", got2)
	}
}

func TestResolveProfile(t *testing.T) {
	def := DefaultConfig()
	p, err := def.ResolveProfile("stage")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.JWTHeader != "jwt-token" {
		t.Errorf("unexpected JWTHeader: %s", p.JWTHeader)
	}

	_, err = def.ResolveProfile("unknown-env")
	if err == nil {
		t.Error("expected error for unknown env")
	}
}

func TestFindSubgraph(t *testing.T) {
	def := DefaultConfig()
	sg, ok := def.FindSubgraph("discovery")
	if !ok {
		t.Fatal("expected to find discovery subgraph")
	}
	if sg.Namespace != "pl-discovery" {
		t.Errorf("unexpected namespace: %s", sg.Namespace)
	}

	_, ok = def.FindSubgraph("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent subgraph")
	}
}
