package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		expectedURL string
	}{
		{
			name:        "with custom URL",
			baseURL:     "https://custom.api.com/",
			expectedURL: "https://custom.api.com/",
		},
		{
			name:        "with empty URL uses default",
			baseURL:     "",
			expectedURL: DefaultAPIHost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL)
			if client.BaseURL != tt.expectedURL {
				t.Errorf("Expected BaseURL %s, got %s", tt.expectedURL, client.BaseURL)
			}
			if client.HTTPClient == nil {
				t.Error("HTTPClient should not be nil")
			}
		})
	}
}

func TestFetchK8sResource_Success(t *testing.T) {
	expectedConfig := "base64-encoded-config"
	expectedName := "test-cluster"

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" && authHeader != "test-token" {
			t.Errorf("Missing or incorrect authorization header: %s", authHeader)
		}
		if r.URL.Query().Get("resourceName") != expectedName {
			t.Errorf("Expected resourceName %s, got %s", expectedName, r.URL.Query().Get("resourceName"))
		}

		// Send response
		response := K8sResourceResponse{
			Name:   expectedName,
			Config: expectedConfig,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL + "/")
	config, err := client.FetchK8sResource("test-token", expectedName)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if config != expectedConfig {
		t.Errorf("Expected config %s, got %s", expectedConfig, config)
	}
}

func TestFetchK8sResource_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	client := NewClient(server.URL + "/")
	_, err := client.FetchK8sResource("invalid-token", "test-cluster")

	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
}

func TestFetchK8sResource_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Resource not found"))
	}))
	defer server.Close()

	client := NewClient(server.URL + "/")
	_, err := client.FetchK8sResource("test-token", "non-existent")

	if err == nil {
		t.Error("Expected error for not found resource")
	}
}

func TestFetchK8sResource_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL + "/")
	_, err := client.FetchK8sResource("test-token", "test-cluster")

	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
}

func TestLogin_Success(t *testing.T) {
	expectedToken := "test-jwt-token"
	username := "testuser"
	password := "testpass"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Missing or incorrect Content-Type header")
		}

		var loginReq LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			t.Errorf("Failed to decode login request: %v", err)
		}
		if loginReq.Username != username || loginReq.Password != password {
			t.Errorf("Incorrect credentials in request body")
		}

		response := LoginResponse{
			JWT:      expectedToken,
			Expires:  1766169648,
			IssuedAt: 1766168748,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL + "/")
	loginResp, err := client.Login(username, password)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if loginResp == nil || loginResp.JWT != expectedToken {
		t.Errorf("Expected token %s, got %v", expectedToken, loginResp)
	}
}
