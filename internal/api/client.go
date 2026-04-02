package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}

func (c *Client) decode(resp *http.Response, out any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if json.Unmarshal(body, &errResp) == nil {
			if msg, ok := errResp["error"].(string); ok {
				return fmt.Errorf("%s", msg)
			}
		}
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

// Auth

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (c *Client) Login(email, password string) (*AuthResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/auth/login", LoginRequest{Email: email, Password: password})
	if err != nil {
		return nil, err
	}
	var out AuthResponse
	return &out, c.decode(resp, &out)
}

func (c *Client) Register(email, password, name string) (*AuthResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/auth/register", RegisterRequest{Email: email, Password: password, Name: name})
	if err != nil {
		return nil, err
	}
	var out AuthResponse
	return &out, c.decode(resp, &out)
}

func (c *Client) Profile() (*User, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/auth/profile", nil)
	if err != nil {
		return nil, err
	}
	var out User
	return &out, c.decode(resp, &out)
}

// VMs

type VM struct {
	ID          string `json:"vm_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	VCPUCount   int    `json:"vcpu_count"`
	MemoryMB    int    `json:"memory_mb"`
	IPAddress   string `json:"ip_address"`
}

type CreateVMRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	VCPUCount   int    `json:"vcpu_count"`
	MemoryMB    int    `json:"memory_mb"`
}

type ListVMsResponse struct {
	Items      []VM `json:"items"`
	Total      int  `json:"total"`
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
}

func (c *Client) ListVMs(page, pageSize int) (*ListVMsResponse, error) {
	resp, err := c.do(http.MethodGet, fmt.Sprintf("/api/v1/vms?page=%d&page_size=%d", page, pageSize), nil)
	if err != nil {
		return nil, err
	}
	var out ListVMsResponse
	return &out, c.decode(resp, &out)
}

func (c *Client) GetVM(id string) (*VM, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/vms/"+id, nil)
	if err != nil {
		return nil, err
	}
	var out VM
	return &out, c.decode(resp, &out)
}

func (c *Client) CreateVM(req CreateVMRequest) (*VM, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/vms", req)
	if err != nil {
		return nil, err
	}
	var out VM
	return &out, c.decode(resp, &out)
}

func (c *Client) DeleteVM(id string) error {
	resp, err := c.do(http.MethodDelete, "/api/v1/vms/"+id, nil)
	if err != nil {
		return err
	}
	return c.decode(resp, nil)
}

func (c *Client) StartVM(id string) error {
	resp, err := c.do(http.MethodPost, "/api/v1/vms/"+id+"/start", nil)
	if err != nil {
		return err
	}
	return c.decode(resp, nil)
}

func (c *Client) StopVM(id string) error {
	resp, err := c.do(http.MethodPost, "/api/v1/vms/"+id+"/stop", nil)
	if err != nil {
		return err
	}
	return c.decode(resp, nil)
}

func (c *Client) RestartVM(id string) error {
	resp, err := c.do(http.MethodPost, "/api/v1/vms/"+id+"/restart", nil)
	if err != nil {
		return err
	}
	return c.decode(resp, nil)
}

type DeployVMRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	VCPUCount   int    `json:"vcpu_count"`
	MemoryMB    int    `json:"memory_mb"`
	RepoURL     string `json:"repo_url"`
	Builder     string `json:"builder,omitempty"`
	KernelPath  string `json:"kernel_path,omitempty"`
}

func (c *Client) DeployVM(req DeployVMRequest) (*VM, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/vms/build", req)
	if err != nil {
		return nil, err
	}
	var out VM
	return &out, c.decode(resp, &out)
}

// IP Pools

type IPPool struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CIDR      string `json:"cidr"`
	Gateway   string `json:"gateway"`
	StartIP   string `json:"start_ip"`
	EndIP     string `json:"end_ip"`
}

type CreateIPPoolRequest struct {
	Name    string `json:"name"`
	CIDR    string `json:"cidr"`
	Gateway string `json:"gateway"`
	StartIP string `json:"start_ip"`
	EndIP   string `json:"end_ip"`
}

type ListIPPoolsResponse struct {
	IPPools  []IPPool `json:"ip_pools"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

type IPPoolStats struct {
	Total     int `json:"total"`
	Allocated int `json:"allocated"`
	Available int `json:"available"`
}

func (c *Client) ListIPPools(page, pageSize int) (*ListIPPoolsResponse, error) {
	resp, err := c.do(http.MethodGet, fmt.Sprintf("/api/v1/ippools?page=%d&page_size=%d", page, pageSize), nil)
	if err != nil {
		return nil, err
	}
	var out ListIPPoolsResponse
	return &out, c.decode(resp, &out)
}

func (c *Client) GetIPPool(id string) (*IPPool, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/ippools/"+id, nil)
	if err != nil {
		return nil, err
	}
	var out IPPool
	return &out, c.decode(resp, &out)
}

func (c *Client) CreateIPPool(req CreateIPPoolRequest) (*IPPool, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/ippools", req)
	if err != nil {
		return nil, err
	}
	var out IPPool
	return &out, c.decode(resp, &out)
}

func (c *Client) DeleteIPPool(id string) error {
	resp, err := c.do(http.MethodDelete, "/api/v1/ippools/"+id, nil)
	if err != nil {
		return err
	}
	return c.decode(resp, nil)
}

func (c *Client) GetIPPoolStats(id string) (*IPPoolStats, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/ippools/"+id+"/stats", nil)
	if err != nil {
		return nil, err
	}
	var out IPPoolStats
	return &out, c.decode(resp, &out)
}
