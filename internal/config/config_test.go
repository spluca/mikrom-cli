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

func TestSave_SyncsActiveContext(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	cfg := &Config{
		APIURL:         "http://new-url:8080",
		Token:          "new-token",
		CurrentContext: "prod",
		Contexts:       map[string]ContextEntry{"prod": {APIURL: "http://old:8080", Token: "old"}},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.APIURL != "http://new-url:8080" {
		t.Errorf("APIURL: got %q, want http://new-url:8080", loaded.APIURL)
	}
	if loaded.Token != "new-token" {
		t.Errorf("Token: got %q, want new-token", loaded.Token)
	}
	entry := loaded.Contexts["prod"]
	if entry.APIURL != "http://new-url:8080" || entry.Token != "new-token" {
		t.Errorf("context entry not synced: %+v", entry)
	}
}

func TestLoad_SyncsActiveContext(t *testing.T) {
	tmp := t.TempDir()
	setHome(t, tmp)

	raw := Config{
		CurrentContext: "staging",
		Contexts: map[string]ContextEntry{
			"staging": {APIURL: "http://staging:9000", Token: "stg-token"},
		},
	}
	data, _ := json.Marshal(raw)
	dir := filepath.Join(tmp, ".mikrom")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.APIURL != "http://staging:9000" {
		t.Errorf("APIURL not synced from context: got %q", cfg.APIURL)
	}
	if cfg.Token != "stg-token" {
		t.Errorf("Token not synced from context: got %q", cfg.Token)
	}
}

// --- ActiveContext ---

func TestActiveContext_Default(t *testing.T) {
	cfg := &Config{}
	if got := cfg.ActiveContext(); got != "default" {
		t.Errorf("ActiveContext(): got %q, want %q", got, "default")
	}
}

func TestActiveContext_WithContext(t *testing.T) {
	cfg := &Config{CurrentContext: "prod"}
	if got := cfg.ActiveContext(); got != "prod" {
		t.Errorf("ActiveContext(): got %q, want %q", got, "prod")
	}
}

// --- AddContext ---

func TestAddContext_New(t *testing.T) {
	cfg := &Config{}
	cfg.AddContext("dev", "http://dev:8080", "dev-token")

	entry, ok := cfg.Contexts["dev"]
	if !ok {
		t.Fatal("context 'dev' not found after AddContext")
	}
	if entry.APIURL != "http://dev:8080" || entry.Token != "dev-token" {
		t.Errorf("unexpected entry: %+v", entry)
	}
}

func TestAddContext_Overwrite(t *testing.T) {
	cfg := &Config{
		Contexts: map[string]ContextEntry{
			"dev": {APIURL: "http://old:8080", Token: "old-token"},
		},
	}
	cfg.AddContext("dev", "http://new:9090", "new-token")

	entry := cfg.Contexts["dev"]
	if entry.APIURL != "http://new:9090" || entry.Token != "new-token" {
		t.Errorf("context not overwritten: %+v", entry)
	}
}

func TestAddContext_InitializesMap(t *testing.T) {
	cfg := &Config{}
	cfg.AddContext("x", "http://x:1", "tok")
	if cfg.Contexts == nil {
		t.Error("Contexts map should not be nil after AddContext")
	}
}

// --- UseContext ---

func TestUseContext_Success(t *testing.T) {
	cfg := &Config{
		APIURL:         "http://old:8080",
		Token:          "old-token",
		CurrentContext: "dev",
		Contexts: map[string]ContextEntry{
			"dev":  {APIURL: "http://old:8080", Token: "old-token"},
			"prod": {APIURL: "http://prod:8080", Token: "prod-token"},
		},
	}

	if err := cfg.UseContext("prod"); err != nil {
		t.Fatalf("UseContext() error: %v", err)
	}
	if cfg.CurrentContext != "prod" {
		t.Errorf("CurrentContext: got %q, want prod", cfg.CurrentContext)
	}
	if cfg.APIURL != "http://prod:8080" {
		t.Errorf("APIURL: got %q, want http://prod:8080", cfg.APIURL)
	}
	if cfg.Token != "prod-token" {
		t.Errorf("Token: got %q, want prod-token", cfg.Token)
	}
}

func TestUseContext_PersistsOutgoing(t *testing.T) {
	cfg := &Config{
		APIURL:         "http://modified:8080",
		Token:          "modified-token",
		CurrentContext: "dev",
		Contexts: map[string]ContextEntry{
			"dev":  {APIURL: "http://original:8080", Token: "original-token"},
			"prod": {APIURL: "http://prod:8080", Token: "prod-token"},
		},
	}

	cfg.UseContext("prod")

	// The outgoing "dev" context should have been updated with the in-memory values.
	dev := cfg.Contexts["dev"]
	if dev.APIURL != "http://modified:8080" || dev.Token != "modified-token" {
		t.Errorf("outgoing context not persisted: %+v", dev)
	}
}

func TestUseContext_NotFound(t *testing.T) {
	cfg := &Config{
		Contexts: map[string]ContextEntry{
			"dev": {APIURL: "http://dev:8080", Token: "tok"},
		},
	}
	if err := cfg.UseContext("missing"); err == nil {
		t.Error("expected error for missing context, got nil")
	}
}

func TestUseContext_NilContexts(t *testing.T) {
	cfg := &Config{}
	if err := cfg.UseContext("any"); err == nil {
		t.Error("expected error when Contexts is nil, got nil")
	}
}

// --- RemoveContext ---

func TestRemoveContext_Success(t *testing.T) {
	cfg := &Config{
		CurrentContext: "dev",
		Contexts: map[string]ContextEntry{
			"dev":  {APIURL: "http://dev:8080", Token: "tok"},
			"prod": {APIURL: "http://prod:8080", Token: "tok2"},
		},
	}
	if err := cfg.RemoveContext("prod"); err != nil {
		t.Fatalf("RemoveContext() error: %v", err)
	}
	if _, ok := cfg.Contexts["prod"]; ok {
		t.Error("expected 'prod' to be removed")
	}
}

func TestRemoveContext_ActiveError(t *testing.T) {
	cfg := &Config{
		CurrentContext: "prod",
		Contexts: map[string]ContextEntry{
			"prod": {APIURL: "http://prod:8080", Token: "tok"},
		},
	}
	if err := cfg.RemoveContext("prod"); err == nil {
		t.Error("expected error when removing active context, got nil")
	}
}

func TestRemoveContext_NotFound(t *testing.T) {
	cfg := &Config{
		Contexts: map[string]ContextEntry{
			"dev": {APIURL: "http://dev:8080", Token: "tok"},
		},
	}
	if err := cfg.RemoveContext("missing"); err == nil {
		t.Error("expected error for missing context, got nil")
	}
}

func TestRemoveContext_NilContexts(t *testing.T) {
	cfg := &Config{CurrentContext: "other"}
	if err := cfg.RemoveContext("missing"); err == nil {
		t.Error("expected error when Contexts is nil, got nil")
	}
}
