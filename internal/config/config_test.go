package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setHome overrides the HOME env var for the duration of a test.
func setHome(t *testing.T, dir string) {
	t.Helper()
	original := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Setenv("HOME", original) })
}

func TestLoad_FileNotExist(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("expected default APIURL, got %q", cfg.APIURL)
	}
	if cfg.Token != "" {
		t.Errorf("expected empty token, got %q", cfg.Token)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	want := Config{APIURL: "http://example.com:9090", Token: "tok123"}
	data, _ := json.Marshal(want)
	dir := filepath.Join(tmp, ".mikrom")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIURL != want.APIURL {
		t.Errorf("APIURL: got %q, want %q", cfg.APIURL, want.APIURL)
	}
	if cfg.Token != want.Token {
		t.Errorf("Token: got %q, want %q", cfg.Token, want.Token)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	dir := filepath.Join(tmp, ".mikrom")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "config.json"), []byte("not-json"), 0600)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestSave(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	cfg := &Config{APIURL: "http://save-test:8080", Token: "saved-token"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path := filepath.Join(tmp, ".mikrom", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if got.APIURL != cfg.APIURL {
		t.Errorf("APIURL: got %q, want %q", got.APIURL, cfg.APIURL)
	}
	if got.Token != cfg.Token {
		t.Errorf("Token: got %q, want %q", got.Token, cfg.Token)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	// Ensure .mikrom dir does NOT exist yet
	dir := filepath.Join(tmp, ".mikrom")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("expected .mikrom dir to not exist")
	}

	cfg := &Config{APIURL: "http://localhost:8080", Token: ""}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("expected .mikrom dir to be created")
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	original := &Config{APIURL: "http://roundtrip:1234", Token: "round-token"}
	if err := original.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.APIURL != original.APIURL {
		t.Errorf("APIURL: got %q, want %q", loaded.APIURL, original.APIURL)
	}
	if loaded.Token != original.Token {
		t.Errorf("Token: got %q, want %q", loaded.Token, original.Token)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	cfg := &Config{APIURL: "http://localhost:8080", Token: "secret"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path := filepath.Join(tmp, ".mikrom", "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions: got %o, want 0600", perm)
	}
}
