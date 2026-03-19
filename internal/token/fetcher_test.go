package token

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fponin/hpub/internal/ui"
)

func init() {
	// Suppress UI output in tests
	ui.SetNoColor(true)
}

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testtoken" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"jwtToken": "eyJhbGciOiJIUzI1NiJ9.test"})
	}))
	defer srv.Close()

	tok, err := Fetch(context.Background(), srv.URL, "testtoken")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(tok, "eyJ") {
		t.Errorf("unexpected token: %s", tok)
	}
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, "tok")
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestFetch_MissingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"other": "value"})
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, "tok")
	if err == nil {
		t.Fatal("expected error for missing jwtToken")
	}
}

func TestFetch_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, "tok")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetch_Unreachable(t *testing.T) {
	_, err := Fetch(context.Background(), "http://127.0.0.1:19999/auth", "tok")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}
