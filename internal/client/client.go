package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"vibeman/internal/constants"
	"vibeman/internal/xdg"

	"github.com/gorilla/websocket"
)

// Client represents the HTTP/WebSocket client for vibeman server
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	tokenStore *TokenStore
}

// New creates a new client instance
func New(serverURL string) (*Client, error) {
	// Parse and validate the server URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Ensure the URL has a scheme
	if u.Scheme == "" {
		u.Scheme = "http"
	}

	tokenStore, err := NewTokenStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create token store: %w", err)
	}

	c := &Client{
		baseURL: u.String(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokenStore: tokenStore,
	}

	// Load existing token if available
	token, err := tokenStore.Load()
	if err == nil && token != "" {
		c.token = token
	}

	return c, nil
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) error {
	c.token = token
	return c.tokenStore.Save(token)
}

// ClearToken clears the authentication token
func (c *Client) ClearToken() error {
	c.token = ""
	return c.tokenStore.Clear()
}

// IsAuthenticated checks if the client has a valid token
func (c *Client) IsAuthenticated() bool {
	return c.token != ""
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for authentication errors
	if resp.StatusCode == http.StatusUnauthorized {
		// Try to refresh token
		if err := c.refreshToken(ctx); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}

		// Retry the request with new token
		req.Header.Set("Authorization", "Bearer "+c.token)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed after token refresh: %w", err)
		}
	}

	return resp, nil
}

// refreshToken attempts to refresh the authentication token
func (c *Client) refreshToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/auth/refresh", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed: %s", resp.Status)
	}

	var result struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	return c.SetToken(result.Token)
}

// WebSocketConnect establishes a WebSocket connection for real-time operations
func (c *Client) WebSocketConnect(ctx context.Context, path string) (*websocket.Conn, error) {
	// Parse the base URL to get the host
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	// Build WebSocket URL
	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s%s", wsScheme, u.Host, path)

	// Add authentication header
	header := http.Header{}
	if c.token != "" {
		header.Set("Authorization", "Bearer "+c.token)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("WebSocket connection failed: %w", err)
	}

	return conn, nil
}

// TokenStore handles persistent storage of authentication tokens
type TokenStore struct {
	path string
}

// NewTokenStore creates a new token store
func NewTokenStore() (*TokenStore, error) {
	configDir, err := xdg.ConfigDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(configDir, constants.SecureDirPermissions); err != nil {
		return nil, err
	}

	return &TokenStore{
		path: filepath.Join(configDir, "token"),
	}, nil
}

// Save saves the token to disk
func (ts *TokenStore) Save(token string) error {
	return os.WriteFile(ts.path, []byte(token), constants.SecureFilePermissions)
}

// Load loads the token from disk
func (ts *TokenStore) Load() (string, error) {
	data, err := os.ReadFile(ts.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Clear removes the stored token
func (ts *TokenStore) Clear() error {
	err := os.Remove(ts.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Health checks the health of the server
func (c *Client) Health(ctx context.Context) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed: %s", resp.Status)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return health, nil
}
