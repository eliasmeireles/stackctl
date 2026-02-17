package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var DefaultAPIHost = "http://contabo.solutionstk.network:31000/api/oauth/v1/"

// LoginRequest represents the login API request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login API response
type LoginResponse struct {
	JWT      string `json:"jwt"`
	Expires  int64  `json:"expires"`
	IssuedAt int64  `json:"issuedAt"`
}

// K8sResourceRequest represents the request body for saving/updating
type K8sResourceRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// K8sResourceResponse represents the API response
type K8sResourceResponse struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// Client represents an API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultAPIHost
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchK8sResource fetches the K8s resource config from the API
func (c *Client) FetchK8sResource(authToken, name string) (string, error) {
	requestURL, err := url.JoinPath(c.BaseURL, "resources/k8s")
	if err != nil {
		return "", fmt.Errorf("failed to join URL: %w", err)
	}

	u, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("resourceName", name)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var resource K8sResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return resource.Config, nil
}

// Login authenticates with the API and returns a LoginResponse
func (c *Client) Login(username, password string) (*LoginResponse, error) {
	url, err := url.JoinPath(c.BaseURL, "login")

	if err != nil {
		return nil, fmt.Errorf("failed to join URL: %w", err)
	}

	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}
	body, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	if loginResp.JWT == "" {
		return nil, fmt.Errorf("login failed: token is empty")
	}

	return &loginResp, nil
}

// AddK8sResource adds a new K8s resource to the remote API
func (c *Client) AddK8sResource(authToken, name, config string) error {
	return c.sendK8sResourceRequest("POST", authToken, name, config)
}

// UpdateK8sResource updates an existing K8s resource in the remote API
func (c *Client) UpdateK8sResource(authToken, name, config string) error {
	return c.sendK8sResourceRequest("PUT", authToken, name, config)
}

func (c *Client) sendK8sResourceRequest(method, authToken, name, config string) error {
	url, err := url.JoinPath(c.BaseURL, "resources/k8s")
	if err != nil {
		return fmt.Errorf("failed to join URL: %w", err)
	}

	resourceReq := K8sResourceRequest{
		Name:   name,
		Config: config,
	}
	body, err := json.Marshal(resourceReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(method, url, io.NopCloser(bytes.NewBuffer(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
