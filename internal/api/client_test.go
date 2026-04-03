package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer creates a test HTTP server that responds with the given status
// code and body (marshaled from v, or raw string if v is a string).
func newTestServer(t *testing.T, status int, v any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		switch body := v.(type) {
		case string:
			w.Write([]byte(body))
		default:
			json.NewEncoder(w).Encode(body)
		}
	}))
}

// newTestServerFunc creates a test server with a custom handler.
func newTestServerFunc(t *testing.T, fn http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(fn)
}

// --- NewClient ---

func TestNewClient(t *testing.T) {
	c := NewClient("http://example.com", "mytoken")
	if c.baseURL != "http://example.com" {
		t.Errorf("baseURL: got %q, want %q", c.baseURL, "http://example.com")
	}
	if c.token != "mytoken" {
		t.Errorf("token: got %q, want %q", c.token, "mytoken")
	}
	if c.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

// --- decode ---

func TestDecode_ErrorWithMessage(t *testing.T) {
	srv := newTestServer(t, http.StatusUnauthorized, map[string]any{"error": "invalid credentials"})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, _ := c.httpClient.Get(srv.URL)
	err := c.decode(resp, nil)
	if err == nil || err.Error() != "invalid credentials" {
		t.Errorf("expected 'invalid credentials' error, got %v", err)
	}
}

func TestDecode_ErrorWithoutMessage(t *testing.T) {
	srv := newTestServer(t, http.StatusInternalServerError, `{}`)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, _ := c.httpClient.Get(srv.URL)
	err := c.decode(resp, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDecode_Success_NilOut(t *testing.T) {
	srv := newTestServer(t, http.StatusOK, `{}`)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, _ := c.httpClient.Get(srv.URL)
	if err := c.decode(resp, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- do: Authorization header ---

func TestDo_SetsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok-xyz")
	c.do(http.MethodGet, "/", nil)

	if gotAuth != "Bearer tok-xyz" {
		t.Errorf("Authorization header: got %q, want %q", gotAuth, "Bearer tok-xyz")
	}
}

func TestDo_NoAuthHeaderWhenEmptyToken(t *testing.T) {
	var gotAuth string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	c.do(http.MethodGet, "/", nil)

	if gotAuth != "" {
		t.Errorf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestDo_ContentTypeHeader(t *testing.T) {
	var gotCT string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	c.do(http.MethodPost, "/", map[string]string{"key": "val"})

	if gotCT != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", gotCT)
	}
}

// --- Auth ---

func TestLogin_Success(t *testing.T) {
	want := AuthResponse{Token: "jwt-token", User: User{ID: 1, Email: "a@b.com", Name: "Alice"}}
	srv := newTestServer(t, http.StatusOK, want)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	got, err := c.Login("a@b.com", "pass")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if got.Token != want.Token {
		t.Errorf("Token: got %q, want %q", got.Token, want.Token)
	}
	if got.User.Email != want.User.Email {
		t.Errorf("User.Email: got %q, want %q", got.User.Email, want.User.Email)
	}
}

func TestLogin_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusUnauthorized, map[string]any{"error": "invalid credentials"})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Login("a@b.com", "wrong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLogin_SendsCorrectBody(t *testing.T) {
	var got LoginRequest
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		json.NewEncoder(w).Encode(AuthResponse{Token: "t"})
	})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	c.Login("user@example.com", "s3cret")

	if got.Email != "user@example.com" || got.Password != "s3cret" {
		t.Errorf("unexpected body: %+v", got)
	}
}

func TestRegister_Success(t *testing.T) {
	want := RegisterResponse{Message: "User created successfully", User: User{ID: 2, Email: "b@c.com", Name: "Bob"}}
	srv := newTestServer(t, http.StatusCreated, want)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	got, err := c.Register("b@c.com", "pass", "Bob")
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	if got.User.Name != "Bob" {
		t.Errorf("Name: got %q, want %q", got.User.Name, "Bob")
	}
	if got.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestRegister_SendsCorrectBody(t *testing.T) {
	var got RegisterRequest
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		json.NewEncoder(w).Encode(RegisterResponse{Message: "ok"})
	})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	c.Register("x@y.com", "pwd", "Xavier")

	if got.Email != "x@y.com" || got.Password != "pwd" || got.Name != "Xavier" {
		t.Errorf("unexpected body: %+v", got)
	}
}

func TestProfile_Success(t *testing.T) {
	want := User{ID: 3, Email: "c@d.com", Name: "Carol"}
	srv := newTestServer(t, http.StatusOK, want)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.Profile()
	if err != nil {
		t.Fatalf("Profile() error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %d, want %d", got.ID, want.ID)
	}
}

func TestProfile_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Profile()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- VMs ---

func TestListVMs_Success(t *testing.T) {
	want := ListVMsResponse{
		Items:    []VM{{ID: "vm1", Name: "test-vm", Status: "running"}},
		Total:    1,
		Page:     1,
		PageSize: 10,
	}
	var gotURL string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.ListVMs(1, 10, "")
	if err != nil {
		t.Fatalf("ListVMs() error: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "vm1" {
		t.Errorf("unexpected Items: %+v", got.Items)
	}
	if gotURL != "/api/v1/vms?page=1&page_size=10" {
		t.Errorf("unexpected URL: %q", gotURL)
	}
}

func TestListVMs_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusInternalServerError, map[string]any{"error": "server error"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.ListVMs(1, 10, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetVM_Success(t *testing.T) {
	want := VM{ID: "vm-abc", Name: "my-vm", Status: "stopped", VCPUCount: 2, MemoryMB: 512}
	srv := newTestServer(t, http.StatusOK, want)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.GetVM("vm-abc")
	if err != nil {
		t.Fatalf("GetVM() error: %v", err)
	}
	if got.ID != want.ID || got.VCPUCount != 2 {
		t.Errorf("unexpected VM: %+v", got)
	}
}

func TestGetVM_NotFound(t *testing.T) {
	srv := newTestServer(t, http.StatusNotFound, map[string]any{"error": "not found"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.GetVM("missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateVM_Success(t *testing.T) {
	want := VM{ID: "new-vm", Name: "fresh", Status: "pending", VCPUCount: 1, MemoryMB: 256}
	var gotBody CreateVMRequest
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	req := CreateVMRequest{Name: "fresh", VCPUCount: 1, MemoryMB: 256}
	c := NewClient(srv.URL, "tok")
	got, err := c.CreateVM(req)
	if err != nil {
		t.Fatalf("CreateVM() error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
	if gotBody.Name != "fresh" || gotBody.VCPUCount != 1 {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
}

func TestUpdateVM_Success(t *testing.T) {
	name := "renamed"
	want := VM{ID: "vm-abc", Name: "renamed", Status: "running"}
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.UpdateVM("vm-abc", UpdateVMRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateVM() error: %v", err)
	}
	if got.Name != "renamed" {
		t.Errorf("Name: got %q, want renamed", got.Name)
	}
	if gotPath != "/api/v1/vms/vm-abc" {
		t.Errorf("path: got %q, want /api/v1/vms/vm-abc", gotPath)
	}
}

func TestDeleteVM_Success(t *testing.T) {
	srv := newTestServer(t, http.StatusNoContent, nil)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if err := c.DeleteVM("vm1"); err != nil {
		t.Errorf("DeleteVM() error: %v", err)
	}
}

func TestDeleteVM_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusNotFound, map[string]any{"error": "not found"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if err := c.DeleteVM("missing"); err == nil {
		t.Error("expected error, got nil")
	}
}

func testVMAction(t *testing.T, action func(c *Client, id string) error, actionName string) {
	t.Helper()
	t.Run(actionName+"_success", func(t *testing.T) {
		srv := newTestServer(t, http.StatusAccepted, `{}`)
		defer srv.Close()
		c := NewClient(srv.URL, "tok")
		if err := action(c, "vm1"); err != nil {
			t.Errorf("%s() error: %v", actionName, err)
		}
	})
	t.Run(actionName+"_error", func(t *testing.T) {
		srv := newTestServer(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid state"})
		defer srv.Close()
		c := NewClient(srv.URL, "tok")
		if err := action(c, "vm1"); err == nil {
			t.Errorf("expected error from %s(), got nil", actionName)
		}
	})
}

func TestStartVM(t *testing.T) {
	testVMAction(t, func(c *Client, id string) error { return c.StartVM(id) }, "StartVM")
}

func TestStopVM(t *testing.T) {
	testVMAction(t, func(c *Client, id string) error { return c.StopVM(id) }, "StopVM")
}

func TestRestartVM(t *testing.T) {
	testVMAction(t, func(c *Client, id string) error { return c.RestartVM(id) }, "RestartVM")
}

func TestStartVM_UsesCorrectPath(t *testing.T) {
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	c.StartVM("vm-xyz")
	if gotPath != "/api/v1/vms/vm-xyz/start" {
		t.Errorf("path: got %q, want /api/v1/vms/vm-xyz/start", gotPath)
	}
}

func TestStopVM_UsesCorrectPath(t *testing.T) {
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	c.StopVM("vm-xyz")
	if gotPath != "/api/v1/vms/vm-xyz/stop" {
		t.Errorf("path: got %q, want /api/v1/vms/vm-xyz/stop", gotPath)
	}
}

func TestRestartVM_UsesCorrectPath(t *testing.T) {
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	c.RestartVM("vm-xyz")
	if gotPath != "/api/v1/vms/vm-xyz/restart" {
		t.Errorf("path: got %q, want /api/v1/vms/vm-xyz/restart", gotPath)
	}
}

// --- IP Pools ---

func TestListIPPools_Success(t *testing.T) {
	want := ListIPPoolsResponse{
		Items:    []IPPool{{ID: 1, Name: "main-pool", CIDR: "10.0.0.0/24"}},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}
	var gotURL string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.ListIPPools(1, 20)
	if err != nil {
		t.Fatalf("ListIPPools() error: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != 1 {
		t.Errorf("unexpected Items: %+v", got.Items)
	}
	if gotURL != "/api/v1/ippools?page=1&page_size=20" {
		t.Errorf("unexpected URL: %q", gotURL)
	}
}

func TestGetIPPool_Success(t *testing.T) {
	want := IPPool{ID: 1, Name: "test-pool", CIDR: "192.168.1.0/24", Gateway: "192.168.1.1"}
	srv := newTestServer(t, http.StatusOK, want)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.GetIPPool(1)
	if err != nil {
		t.Fatalf("GetIPPool() error: %v", err)
	}
	if got.CIDR != want.CIDR {
		t.Errorf("CIDR: got %q, want %q", got.CIDR, want.CIDR)
	}
}

func TestGetIPPool_NotFound(t *testing.T) {
	srv := newTestServer(t, http.StatusNotFound, map[string]any{"error": "not found"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.GetIPPool(999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateIPPool_Success(t *testing.T) {
	want := IPPool{ID: 5, Name: "prod-pool", CIDR: "10.1.0.0/24"}
	var gotBody CreateIPPoolRequest
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	req := CreateIPPoolRequest{Name: "prod-pool", Network: "10.1.0.0", CIDR: "10.1.0.0/24", Gateway: "10.1.0.1", StartIP: "10.1.0.10", EndIP: "10.1.0.200"}
	c := NewClient(srv.URL, "tok")
	got, err := c.CreateIPPool(req)
	if err != nil {
		t.Fatalf("CreateIPPool() error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %d, want %d", got.ID, want.ID)
	}
	if gotBody.CIDR != "10.1.0.0/24" {
		t.Errorf("unexpected body: %+v", gotBody)
	}
}

func TestUpdateIPPool_Success(t *testing.T) {
	name := "updated-pool"
	want := IPPool{ID: 1, Name: "updated-pool"}
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.UpdateIPPool(1, UpdateIPPoolRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateIPPool() error: %v", err)
	}
	if got.Name != "updated-pool" {
		t.Errorf("Name: got %q, want updated-pool", got.Name)
	}
	if gotPath != "/api/v1/ippools/1" {
		t.Errorf("path: got %q, want /api/v1/ippools/1", gotPath)
	}
}

func TestDeleteIPPool_Success(t *testing.T) {
	srv := newTestServer(t, http.StatusNoContent, nil)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if err := c.DeleteIPPool(1); err != nil {
		t.Errorf("DeleteIPPool() error: %v", err)
	}
}

func TestDeleteIPPool_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusConflict, map[string]any{"error": "pool in use"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if err := c.DeleteIPPool(1); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetIPPoolStats_Success(t *testing.T) {
	want := IPPoolStatsResponse{PoolID: 1, PoolName: "main", Total: 100, Allocated: 30, Available: 70, UsagePercent: 30.0}
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.GetIPPoolStats(1)
	if err != nil {
		t.Fatalf("GetIPPoolStats() error: %v", err)
	}
	if got.Total != 100 || got.Allocated != 30 || got.Available != 70 {
		t.Errorf("unexpected stats: %+v", got)
	}
	if gotPath != "/api/v1/ippools/1/stats" {
		t.Errorf("path: got %q, want /api/v1/ippools/1/stats", gotPath)
	}
}

func TestGetIPPoolStats_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusNotFound, map[string]any{"error": "pool not found"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.GetIPPoolStats(999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetAllPoolStats_Success(t *testing.T) {
	want := map[string]any{
		"pools": []IPPoolStatsResponse{
			{PoolID: 1, PoolName: "main", Total: 100, Allocated: 30, Available: 70, UsagePercent: 30.0},
			{PoolID: 2, PoolName: "dev", Total: 50, Allocated: 5, Available: 45, UsagePercent: 10.0},
		},
	}
	srv := newTestServer(t, http.StatusOK, want)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.GetAllPoolStats()
	if err != nil {
		t.Fatalf("GetAllPoolStats() error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 pools, got %d", len(got))
	}
}

func TestSuggestIPRange_Success(t *testing.T) {
	want := SuggestIPRangeResponse{
		CIDR:           "10.0.0.0/24",
		SuggestedStart: "10.0.0.11",
		SuggestedEnd:   "10.0.0.240",
	}
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.SuggestIPRange("10.0.0.0/24")
	if err != nil {
		t.Fatalf("SuggestIPRange() error: %v", err)
	}
	if got.SuggestedStart != "10.0.0.11" {
		t.Errorf("SuggestedStart: got %q, want 10.0.0.11", got.SuggestedStart)
	}
	if gotPath != "/api/v1/ippools/suggest-range" {
		t.Errorf("path: got %q, want /api/v1/ippools/suggest-range", gotPath)
	}
}

// --- DeployVM ---

func TestDeployVM_Success(t *testing.T) {
	want := VM{ID: "deploy-1", Name: "my-app", Status: "building", VCPUCount: 2, MemoryMB: 1024}
	var gotBody DeployVMRequest
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(want)
	})
	defer srv.Close()

	req := DeployVMRequest{Name: "my-app", RepoURL: "https://github.com/example/app", VCPUCount: 2, MemoryMB: 1024}
	c := NewClient(srv.URL, "tok")
	got, err := c.DeployVM(req)
	if err != nil {
		t.Fatalf("DeployVM() error: %v", err)
	}
	if got.ID != want.ID || got.Status != "building" {
		t.Errorf("unexpected VM: %+v", got)
	}
	if gotBody.Name != "my-app" || gotBody.RepoURL != "https://github.com/example/app" {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
}

func TestDeployVM_UsesCorrectPath(t *testing.T) {
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(VM{})
	})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	c.DeployVM(DeployVMRequest{Name: "app", RepoURL: "https://github.com/x/y"}) //nolint:errcheck

	if gotPath != "/api/v1/vms/build" {
		t.Errorf("path: got %q, want /api/v1/vms/build", gotPath)
	}
}

func TestDeployVM_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusUnprocessableEntity, map[string]any{"error": "invalid repo"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.DeployVM(DeployVMRequest{Name: "bad", RepoURL: "not-a-url"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- do: network error ---

func TestDo_NetworkError(t *testing.T) {
	// Use a port where nothing is listening to trigger a connection error.
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.do(http.MethodGet, "/", nil)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestLogin_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "")
	_, err := c.Login("a@b.com", "pass")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestRegister_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "")
	_, err := c.Register("a@b.com", "pass", "Name")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestProfile_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.Profile()
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestHealth_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "")
	_, err := c.Health()
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestListVMs_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.ListVMs(1, 10, "")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestGetVM_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.GetVM("vm1")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestCreateVM_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.CreateVM(CreateVMRequest{Name: "x", VCPUCount: 1, MemoryMB: 256})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestUpdateVM_NetworkError(t *testing.T) {
	name := "x"
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.UpdateVM("vm1", UpdateVMRequest{Name: &name})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestDeleteVM_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	if err := c.DeleteVM("vm1"); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestStartVM_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	if err := c.StartVM("vm1"); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestStopVM_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	if err := c.StopVM("vm1"); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestRestartVM_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	if err := c.RestartVM("vm1"); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestDeployVM_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.DeployVM(DeployVMRequest{Name: "app", RepoURL: "https://github.com/x/y"})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestListIPPools_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.ListIPPools(1, 10)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestGetIPPool_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.GetIPPool(1)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestCreateIPPool_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.CreateIPPool(CreateIPPoolRequest{Name: "x", CIDR: "10.0.0.0/24"})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestUpdateIPPool_NetworkError(t *testing.T) {
	name := "x"
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.UpdateIPPool(1, UpdateIPPoolRequest{Name: &name})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestDeleteIPPool_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	if err := c.DeleteIPPool(1); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestGetIPPoolStats_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.GetIPPoolStats(1)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestGetAllPoolStats_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.GetAllPoolStats()
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestSuggestIPRange_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "tok")
	_, err := c.SuggestIPRange("10.0.0.0/24")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// --- Health ---

func TestHealth_Success(t *testing.T) {
	srv := newTestServer(t, http.StatusOK, HealthResponse{Status: "ok"})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	got, err := c.Health()
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if got.Status != "ok" {
		t.Errorf("Status: got %q, want ok", got.Status)
	}
}

func TestHealth_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusServiceUnavailable, map[string]any{"error": "service unavailable"})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Health()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealth_UsesCorrectPath(t *testing.T) {
	var gotPath string
	srv := newTestServerFunc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	})
	defer srv.Close()

	c := NewClient(srv.URL, "")
	c.Health()
	if gotPath != "/health" {
		t.Errorf("path: got %q, want /health", gotPath)
	}
}

// --- ListVMs client-side status filter ---

func TestListVMs_WithStatusFilter(t *testing.T) {
	body := ListVMsResponse{
		Items: []VM{
			{ID: "vm1", Status: "running"},
			{ID: "vm2", Status: "stopped"},
			{ID: "vm3", Status: "running"},
		},
		Total: 3,
	}
	srv := newTestServer(t, http.StatusOK, body)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.ListVMs(1, 10, "running")
	if err != nil {
		t.Fatalf("ListVMs() error: %v", err)
	}
	if len(got.Items) != 2 {
		t.Errorf("expected 2 running VMs, got %d", len(got.Items))
	}
	for _, vm := range got.Items {
		if vm.Status != "running" {
			t.Errorf("expected only running VMs, got status %q for %s", vm.Status, vm.ID)
		}
	}
	if got.Total != 2 {
		t.Errorf("Total: got %d, want 2", got.Total)
	}
}

func TestListVMs_StatusFilter_NoMatch(t *testing.T) {
	body := ListVMsResponse{
		Items: []VM{{ID: "vm1", Status: "stopped"}},
		Total: 1,
	}
	srv := newTestServer(t, http.StatusOK, body)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	got, err := c.ListVMs(1, 10, "running")
	if err != nil {
		t.Fatalf("ListVMs() error: %v", err)
	}
	if len(got.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(got.Items))
	}
}

// --- GetAllPoolStats error ---

func TestGetAllPoolStats_Error(t *testing.T) {
	srv := newTestServer(t, http.StatusInternalServerError, map[string]any{"error": "internal error"})
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.GetAllPoolStats()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
