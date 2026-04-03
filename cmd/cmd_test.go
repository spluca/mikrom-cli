package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spluca/mikrom-cli/internal/api"
	"github.com/spluca/mikrom-cli/internal/config"
)

// captureOutput redirects os.Stdout during fn and returns what was written.
func captureOutput(fn func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

// setupCfg sets the package-level cfg to point at the given server and token.
func setupCfg(apiURL, token string) {
	cfg = &config.Config{APIURL: apiURL, Token: token}
}

// jsonServer starts a test HTTP server that returns the given value as JSON
// with the given status code.
func jsonServer(t *testing.T, status int, v any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if v != nil {
			json.NewEncoder(w).Encode(v) //nolint:errcheck
		}
	}))
}

// --- printVM ---

func TestPrintVM(t *testing.T) {
	vm := &api.VM{
		ID:          "abc-123",
		Name:        "my-vm",
		Description: "test machine",
		Status:      "running",
		VCPUCount:   2,
		MemoryMB:    512,
		IPAddress:   "10.0.0.1",
	}

	out := captureOutput(func() { printVM(vm) })

	for _, want := range []string{"abc-123", "my-vm", "test machine", "running", "2", "512", "10.0.0.1"} {
		if !strings.Contains(out, want) {
			t.Errorf("printVM output missing %q; got:\n%s", want, out)
		}
	}
}

// --- printIPPool ---

func TestPrintIPPool(t *testing.T) {
	p := &api.IPPool{
		ID:      1,
		Name:    "prod-pool",
		Network: "10.0.0.0",
		CIDR:    "10.0.0.0/24",
		Gateway: "10.0.0.1",
		StartIP: "10.0.0.10",
		EndIP:   "10.0.0.200",
	}

	out := captureOutput(func() { printIPPool(p) })

	for _, want := range []string{"1", "prod-pool", "10.0.0.0/24", "10.0.0.1", "10.0.0.10", "10.0.0.200"} {
		if !strings.Contains(out, want) {
			t.Errorf("printIPPool output missing %q; got:\n%s", want, out)
		}
	}
}

// --- vm list ---

func TestVMListCmd_NoVMs(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, api.ListVMsResponse{Items: []api.VM{}, Total: 0})
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := vmListCmd.RunE(vmListCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No VMs found") {
		t.Errorf("expected 'No VMs found', got: %s", out)
	}
}

func TestVMListCmd_WithVMs(t *testing.T) {
	body := api.ListVMsResponse{
		Items: []api.VM{{ID: "vm-1", Name: "alpha", Status: "running", VCPUCount: 1, MemoryMB: 256}},
		Total: 1,
	}
	srv := jsonServer(t, http.StatusOK, body)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := vmListCmd.RunE(vmListCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "vm-1") || !strings.Contains(out, "alpha") {
		t.Errorf("expected VM data in output, got: %s", out)
	}
	if !strings.Contains(out, "Total: 1") {
		t.Errorf("expected 'Total: 1' in output, got: %s", out)
	}
}

func TestVMListCmd_APIError(t *testing.T) {
	srv := jsonServer(t, http.StatusInternalServerError, map[string]any{"error": "server error"})
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	err := vmListCmd.RunE(vmListCmd, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- vm get ---

func TestVMGetCmd_Success(t *testing.T) {
	vm := api.VM{ID: "vm-abc", Name: "test", Status: "stopped", VCPUCount: 2, MemoryMB: 512}
	srv := jsonServer(t, http.StatusOK, vm)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := vmGetCmd.RunE(vmGetCmd, []string{"vm-abc"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "vm-abc") {
		t.Errorf("expected vm-abc in output, got: %s", out)
	}
}

func TestVMGetCmd_NotFound(t *testing.T) {
	srv := jsonServer(t, http.StatusNotFound, map[string]any{"error": "not found"})
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	err := vmGetCmd.RunE(vmGetCmd, []string{"missing"})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- vm create ---

func TestVMCreateCmd_Success(t *testing.T) {
	var gotBody api.CreateVMRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint:errcheck
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.VM{ID: "new-vm", Name: "fresh", Status: "pending"}) //nolint:errcheck
	}))
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	vmCreateCmd.Flags().Set("name", "fresh")  //nolint:errcheck
	vmCreateCmd.Flags().Set("vcpus", "2")     //nolint:errcheck
	vmCreateCmd.Flags().Set("memory", "256")  //nolint:errcheck

	out := captureOutput(func() {
		if err := vmCreateCmd.RunE(vmCreateCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "new-vm") {
		t.Errorf("expected 'new-vm' in output, got: %s", out)
	}
	if gotBody.Name != "fresh" {
		t.Errorf("request body name: got %q, want 'fresh'", gotBody.Name)
	}
}

func TestVMCreateCmd_APIError(t *testing.T) {
	srv := jsonServer(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid"})
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	vmCreateCmd.Flags().Set("name", "bad-vm") //nolint:errcheck
	err := vmCreateCmd.RunE(vmCreateCmd, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- vm delete ---

func TestVMDeleteCmd_Success(t *testing.T) {
	srv := jsonServer(t, http.StatusNoContent, nil)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := vmDeleteCmd.RunE(vmDeleteCmd, []string{"vm-del"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "vm-del") {
		t.Errorf("expected vm-del in output, got: %s", out)
	}
}

// --- vm start / stop / restart ---

func TestVMStartCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := jsonServer(t, http.StatusAccepted, map[string]any{})
		defer srv.Close()
		setupCfg(srv.URL, "test-token")

		out := captureOutput(func() {
			if err := vmStartCmd.RunE(vmStartCmd, []string{"vm-1"}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "vm-1") {
			t.Errorf("expected 'vm-1' in output, got: %s", out)
		}
	})
	t.Run("error", func(t *testing.T) {
		srv := jsonServer(t, http.StatusUnprocessableEntity, map[string]any{"error": "bad state"})
		defer srv.Close()
		setupCfg(srv.URL, "test-token")

		if err := vmStartCmd.RunE(vmStartCmd, []string{"vm-1"}); err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestVMStopCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := jsonServer(t, http.StatusAccepted, map[string]any{})
		defer srv.Close()
		setupCfg(srv.URL, "test-token")

		out := captureOutput(func() {
			if err := vmStopCmd.RunE(vmStopCmd, []string{"vm-2"}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "vm-2") {
			t.Errorf("expected 'vm-2' in output, got: %s", out)
		}
	})
	t.Run("error", func(t *testing.T) {
		srv := jsonServer(t, http.StatusUnprocessableEntity, map[string]any{"error": "bad state"})
		defer srv.Close()
		setupCfg(srv.URL, "test-token")

		if err := vmStopCmd.RunE(vmStopCmd, []string{"vm-2"}); err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestVMRestartCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := jsonServer(t, http.StatusAccepted, map[string]any{})
		defer srv.Close()
		setupCfg(srv.URL, "test-token")

		out := captureOutput(func() {
			if err := vmRestartCmd.RunE(vmRestartCmd, []string{"vm-3"}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "vm-3") {
			t.Errorf("expected 'vm-3' in output, got: %s", out)
		}
	})
}

// --- ippool list ---

func TestIPPoolListCmd_NoPools(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, api.ListIPPoolsResponse{Items: []api.IPPool{}, Total: 0})
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := ippoolListCmd.RunE(ippoolListCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No IP pools found") {
		t.Errorf("expected 'No IP pools found', got: %s", out)
	}
}

func TestIPPoolListCmd_WithPools(t *testing.T) {
	body := api.ListIPPoolsResponse{
		Items: []api.IPPool{{ID: 1, Name: "main", CIDR: "10.0.0.0/24", StartIP: "10.0.0.10", EndIP: "10.0.0.200"}},
		Total: 1,
	}
	srv := jsonServer(t, http.StatusOK, body)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := ippoolListCmd.RunE(ippoolListCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "1") || !strings.Contains(out, "main") {
		t.Errorf("expected pool data in output, got: %s", out)
	}
}

// --- ippool get ---

func TestIPPoolGetCmd_Success(t *testing.T) {
	pool := api.IPPool{ID: 1, Name: "test-pool", CIDR: "192.168.1.0/24", Gateway: "192.168.1.1"}
	srv := jsonServer(t, http.StatusOK, pool)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := ippoolGetCmd.RunE(ippoolGetCmd, []string{"1"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "192.168.1.0/24") {
		t.Errorf("expected pool data in output, got: %s", out)
	}
}

func TestIPPoolGetCmd_InvalidID(t *testing.T) {
	setupCfg("http://localhost:8080", "test-token")

	err := ippoolGetCmd.RunE(ippoolGetCmd, []string{"not-a-number"})
	if err == nil {
		t.Error("expected error for non-numeric ID, got nil")
	}
}

// --- ippool create ---

func TestIPPoolCreateCmd_Success(t *testing.T) {
	var gotBody api.CreateIPPoolRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint:errcheck
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.IPPool{ID: 3, Name: "prod"}) //nolint:errcheck
	}))
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	ippoolCreateCmd.Flags().Set("name", "prod")              //nolint:errcheck
	ippoolCreateCmd.Flags().Set("network", "10.1.0.0")       //nolint:errcheck
	ippoolCreateCmd.Flags().Set("cidr", "10.1.0.0/24")       //nolint:errcheck
	ippoolCreateCmd.Flags().Set("gateway", "10.1.0.1")       //nolint:errcheck
	ippoolCreateCmd.Flags().Set("start-ip", "10.1.0.10")     //nolint:errcheck
	ippoolCreateCmd.Flags().Set("end-ip", "10.1.0.200")      //nolint:errcheck

	out := captureOutput(func() {
		if err := ippoolCreateCmd.RunE(ippoolCreateCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "3") {
		t.Errorf("expected pool ID in output, got: %s", out)
	}
	if gotBody.CIDR != "10.1.0.0/24" {
		t.Errorf("CIDR: got %q, want 10.1.0.0/24", gotBody.CIDR)
	}
}

// --- ippool delete ---

func TestIPPoolDeleteCmd_Success(t *testing.T) {
	srv := jsonServer(t, http.StatusNoContent, nil)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := ippoolDeleteCmd.RunE(ippoolDeleteCmd, []string{"1"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "1") {
		t.Errorf("expected pool ID in output, got: %s", out)
	}
}

// --- ippool stats ---

func TestIPPoolStatsCmd_Success(t *testing.T) {
	stats := api.IPPoolStatsResponse{PoolID: 1, PoolName: "main", Total: 100, Allocated: 40, Available: 60, UsagePercent: 40.0}
	srv := jsonServer(t, http.StatusOK, stats)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := ippoolStatsCmd.RunE(ippoolStatsCmd, []string{"1"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "100") || !strings.Contains(out, "40") || !strings.Contains(out, "60") {
		t.Errorf("expected stats in output, got: %s", out)
	}
}

// --- auth logout ---

func TestAuthLogoutCmd(t *testing.T) {
	setupCfg("http://localhost:8080", "existing-token")

	captureOutput(func() {
		if err := authLogoutCmd.RunE(authLogoutCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if cfg.Token != "" {
		t.Errorf("expected empty token after logout, got %q", cfg.Token)
	}
}

// --- auth login ---

func TestAuthLoginCmd_Success(t *testing.T) {
	body := api.AuthResponse{Token: "new-jwt", User: api.User{ID: 1, Name: "Alice", Email: "alice@example.com"}}
	srv := jsonServer(t, http.StatusOK, body)
	defer srv.Close()
	setupCfg(srv.URL, "")

	authLoginCmd.Flags().Set("email", "alice@example.com") //nolint:errcheck
	authLoginCmd.Flags().Set("password", "secret")         //nolint:errcheck

	out := captureOutput(func() {
		if err := authLoginCmd.RunE(authLoginCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if cfg.Token != "new-jwt" {
		t.Errorf("expected token 'new-jwt', got %q", cfg.Token)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "alice@example.com") {
		t.Errorf("expected user info in output, got: %s", out)
	}
}

func TestAuthLoginCmd_APIError(t *testing.T) {
	srv := jsonServer(t, http.StatusUnauthorized, map[string]any{"error": "invalid credentials"})
	defer srv.Close()
	setupCfg(srv.URL, "")

	authLoginCmd.Flags().Set("email", "bad@example.com") //nolint:errcheck
	authLoginCmd.Flags().Set("password", "wrong")        //nolint:errcheck

	err := authLoginCmd.RunE(authLoginCmd, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- auth register ---

func TestAuthRegisterCmd_Success(t *testing.T) {
	body := api.RegisterResponse{Message: "User created successfully", User: api.User{ID: 2, Name: "Bob", Email: "bob@example.com"}}
	srv := jsonServer(t, http.StatusCreated, body)
	defer srv.Close()
	setupCfg(srv.URL, "")

	authRegisterCmd.Flags().Set("name", "Bob")              //nolint:errcheck
	authRegisterCmd.Flags().Set("email", "bob@example.com") //nolint:errcheck
	authRegisterCmd.Flags().Set("password", "pass123")      //nolint:errcheck

	out := captureOutput(func() {
		if err := authRegisterCmd.RunE(authRegisterCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Bob") || !strings.Contains(out, "bob@example.com") {
		t.Errorf("expected user info in output, got: %s", out)
	}
}

// --- auth profile ---

func TestAuthProfileCmd_Success(t *testing.T) {
	user := api.User{ID: 3, Name: "Carol", Email: "carol@example.com"}
	srv := jsonServer(t, http.StatusOK, user)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	out := captureOutput(func() {
		if err := authProfileCmd.RunE(authProfileCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"3", "Carol", "carol@example.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

// --- vm deploy ---

func TestVMDeployCmd_Success(t *testing.T) {
	vm := api.VM{ID: "deploy-1", Name: "my-app", Status: "building", VCPUCount: 2, MemoryMB: 1024}
	srv := jsonServer(t, http.StatusAccepted, vm)
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	vmDeployCmd.Flags().Set("name", "my-app")                          //nolint:errcheck
	vmDeployCmd.Flags().Set("repo", "https://github.com/example/app") //nolint:errcheck

	out := captureOutput(func() {
		if err := vmDeployCmd.RunE(vmDeployCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "deploy-1") {
		t.Errorf("expected 'deploy-1' in output, got: %s", out)
	}
	if !strings.Contains(out, "building") {
		t.Errorf("expected 'building' status in output, got: %s", out)
	}
}

func TestVMDeployCmd_APIError(t *testing.T) {
	srv := jsonServer(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid repo"})
	defer srv.Close()
	setupCfg(srv.URL, "test-token")

	vmDeployCmd.Flags().Set("name", "fail-app")                            //nolint:errcheck
	vmDeployCmd.Flags().Set("repo", "https://github.com/example/bad")     //nolint:errcheck

	err := vmDeployCmd.RunE(vmDeployCmd, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- health ---

func TestHealthCmd_Success(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, api.HealthResponse{Status: "ok"})
	defer srv.Close()
	setupCfg(srv.URL, "")

	out := captureOutput(func() {
		if err := healthCmd.RunE(healthCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "ok") {
		t.Errorf("expected 'ok' in output, got: %s", out)
	}
	if !strings.Contains(out, srv.URL) {
		t.Errorf("expected API URL in output, got: %s", out)
	}
}

func TestHealthCmd_Error(t *testing.T) {
	srv := jsonServer(t, http.StatusServiceUnavailable, map[string]any{"error": "down"})
	defer srv.Close()
	setupCfg(srv.URL, "")

	err := healthCmd.RunE(healthCmd, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- context ---

// setHomeForTest redirects HOME to a temp dir for the duration of the test.
func setHomeForTest(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	original := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	t.Cleanup(func() { os.Setenv("HOME", original) })
}

func TestContextListCmd_NoContexts(t *testing.T) {
	setupCfg("http://localhost:8080", "tok")

	out := captureOutput(func() {
		if err := contextListCmd.RunE(contextListCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "default") {
		t.Errorf("expected 'default' context in output, got: %s", out)
	}
}

func TestContextListCmd_WithContexts(t *testing.T) {
	cfg = &config.Config{
		CurrentContext: "prod",
		APIURL:         "http://prod:8080",
		Token:          "prod-tok",
		Contexts: map[string]config.ContextEntry{
			"prod": {APIURL: "http://prod:8080", Token: "prod-tok"},
			"dev":  {APIURL: "http://dev:8080", Token: "dev-tok"},
		},
	}

	out := captureOutput(func() {
		if err := contextListCmd.RunE(contextListCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "prod") || !strings.Contains(out, "dev") {
		t.Errorf("expected both contexts in output, got: %s", out)
	}
}

func TestContextShowCmd(t *testing.T) {
	cfg = &config.Config{
		CurrentContext: "staging",
		APIURL:         "http://staging:8080",
		Token:          "stg-tok",
	}

	out := captureOutput(func() {
		if err := contextShowCmd.RunE(contextShowCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "staging") {
		t.Errorf("expected 'staging' in output, got: %s", out)
	}
	if !strings.Contains(out, "http://staging:8080") {
		t.Errorf("expected API URL in output, got: %s", out)
	}
}

func TestContextShowCmd_NoAuth(t *testing.T) {
	cfg = &config.Config{APIURL: "http://localhost:8080", Token: ""}

	out := captureOutput(func() {
		if err := contextShowCmd.RunE(contextShowCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "no") {
		t.Errorf("expected 'no' auth in output, got: %s", out)
	}
}

func TestContextAddCmd_Success(t *testing.T) {
	setHomeForTest(t)
	cfg = &config.Config{APIURL: "http://localhost:8080"}

	contextAddCmd.Flags().Set("api-url", "http://prod:9090") //nolint:errcheck
	contextAddCmd.Flags().Set("token", "prod-tok")           //nolint:errcheck

	out := captureOutput(func() {
		if err := contextAddCmd.RunE(contextAddCmd, []string{"prod"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "prod") {
		t.Errorf("expected 'prod' in output, got: %s", out)
	}
	if _, ok := cfg.Contexts["prod"]; !ok {
		t.Error("expected 'prod' context to be added")
	}
}

func TestContextUseCmd_Success(t *testing.T) {
	setHomeForTest(t)
	cfg = &config.Config{
		CurrentContext: "dev",
		APIURL:         "http://dev:8080",
		Token:          "dev-tok",
		Contexts: map[string]config.ContextEntry{
			"dev":  {APIURL: "http://dev:8080", Token: "dev-tok"},
			"prod": {APIURL: "http://prod:8080", Token: "prod-tok"},
		},
	}

	out := captureOutput(func() {
		if err := contextUseCmd.RunE(contextUseCmd, []string{"prod"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "prod") {
		t.Errorf("expected 'prod' in output, got: %s", out)
	}
	if cfg.CurrentContext != "prod" {
		t.Errorf("expected current context to be 'prod', got %q", cfg.CurrentContext)
	}
}

func TestContextUseCmd_NotFound(t *testing.T) {
	setHomeForTest(t)
	cfg = &config.Config{
		Contexts: map[string]config.ContextEntry{
			"dev": {APIURL: "http://dev:8080", Token: "tok"},
		},
	}

	err := contextUseCmd.RunE(contextUseCmd, []string{"missing"})
	if err == nil {
		t.Error("expected error for missing context, got nil")
	}
}

func TestContextRemoveCmd_Success(t *testing.T) {
	setHomeForTest(t)
	cfg = &config.Config{
		CurrentContext: "dev",
		Contexts: map[string]config.ContextEntry{
			"dev":  {APIURL: "http://dev:8080", Token: "tok"},
			"prod": {APIURL: "http://prod:8080", Token: "tok2"},
		},
	}

	out := captureOutput(func() {
		if err := contextRemoveCmd.RunE(contextRemoveCmd, []string{"prod"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "prod") {
		t.Errorf("expected 'prod' in output, got: %s", out)
	}
	if _, ok := cfg.Contexts["prod"]; ok {
		t.Error("expected 'prod' context to be removed")
	}
}

func TestContextRemoveCmd_ActiveError(t *testing.T) {
	setHomeForTest(t)
	cfg = &config.Config{
		CurrentContext: "prod",
		Contexts: map[string]config.ContextEntry{
			"prod": {APIURL: "http://prod:8080", Token: "tok"},
		},
	}

	err := contextRemoveCmd.RunE(contextRemoveCmd, []string{"prod"})
	if err == nil {
		t.Error("expected error when removing active context, got nil")
	}
}

// --- waitForVM / waitForVMDeleted ---

func TestWaitForVM_Timeout(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, api.VM{ID: "vm1", Status: "starting"})
	defer srv.Close()
	c := api.NewClient(srv.URL, "tok")

	// timeout=0 → loop condition false immediately, no sleep
	_, err := waitForVM(c, "vm1", "running", 0)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected 'timed out' in error, got: %v", err)
	}
}

func TestWaitForVM_ErrorState(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, api.VM{ID: "vm1", Status: "error"})
	defer srv.Close()
	c := api.NewClient(srv.URL, "tok")

	// Use a generous timeout so the loop runs at least once (needs 1 poll).
	_, err := waitForVM(c, "vm1", "running", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for VM in error state, got nil")
	}
	if !strings.Contains(err.Error(), "error state") {
		t.Errorf("expected 'error state' in error, got: %v", err)
	}
}

func TestWaitForVM_Success(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, api.VM{ID: "vm1", Status: "running"})
	defer srv.Close()
	c := api.NewClient(srv.URL, "tok")

	vm, err := waitForVM(c, "vm1", "running", 10*time.Second)
	if err != nil {
		t.Fatalf("waitForVM() error: %v", err)
	}
	if vm.Status != "running" {
		t.Errorf("Status: got %q, want running", vm.Status)
	}
}

func TestWaitForVMDeleted_Timeout(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, api.VM{ID: "vm1", Status: "deleting"})
	defer srv.Close()
	c := api.NewClient(srv.URL, "tok")

	err := waitForVMDeleted(c, "vm1", 0)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected 'timed out' in error, got: %v", err)
	}
}

func TestWaitForVMDeleted_Success(t *testing.T) {
	srv := jsonServer(t, http.StatusNotFound, map[string]any{"error": "not found"})
	defer srv.Close()
	c := api.NewClient(srv.URL, "tok")

	err := waitForVMDeleted(c, "vm1", 10*time.Second)
	if err != nil {
		t.Fatalf("waitForVMDeleted() error: %v", err)
	}
}
