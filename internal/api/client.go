package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient represents the HTTP client for vibeman server API
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client instance
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ServiceInfo represents service information from the API
type ServiceInfo struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	ContainerID  string            `json:"container_id,omitempty"`
	RefCount     int               `json:"ref_count"`
	Repositories []string          `json:"repositories"`
	StartTime    *time.Time        `json:"start_time,omitempty"`
	Uptime       string            `json:"uptime,omitempty"`
	HealthError  string            `json:"health_error,omitempty"`
	Config       ServiceConfig     `json:"config"`
}

// ServiceConfig represents service configuration
type ServiceConfig struct {
	ComposeFile string `json:"compose_file"`
	Service     string `json:"service"`
}

// GetServices lists all services
func (c *APIClient) GetServices(ctx context.Context) ([]*ServiceInfo, error) {
	resp, err := c.get(ctx, "/api/services")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Services []*ServiceInfo `json:"services"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Services, nil
}

// GetService gets a specific service
func (c *APIClient) GetService(ctx context.Context, name string) (*ServiceInfo, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/services/%s", name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var service ServiceInfo
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &service, nil
}

// StartService starts a service
func (c *APIClient) StartService(ctx context.Context, name string) error {
	resp, err := c.post(ctx, fmt.Sprintf("/api/services/%s/start", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start service: %s", string(body))
	}

	return nil
}

// StopService stops a service
func (c *APIClient) StopService(ctx context.Context, name string) error {
	resp, err := c.post(ctx, fmt.Sprintf("/api/services/%s/stop", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop service: %s", string(body))
	}

	return nil
}

// RestartService restarts a service
func (c *APIClient) RestartService(ctx context.Context, name string) error {
	resp, err := c.post(ctx, fmt.Sprintf("/api/services/%s/restart", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to restart service: %s", string(body))
	}

	return nil
}

// Internal HTTP methods

func (c *APIClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *APIClient) post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}