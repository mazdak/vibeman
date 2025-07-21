package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"vibeman/internal/types"
)

// Service operations

// StartService starts a service
func (c *Client) StartService(ctx context.Context, name string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/services/%s/start", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to start service: %s", resp.Status)
	}

	return nil
}

// StopService stops a service
func (c *Client) StopService(ctx context.Context, name string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/services/%s/stop", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to stop service: %s", resp.Status)
	}

	return nil
}

// AddServiceReference adds a repository reference to a service
func (c *Client) AddServiceReference(ctx context.Context, serviceName, repositoryName string) error {
	req := struct {
		RepositoryName string `json:"projectName"`
	}{
		RepositoryName: repositoryName,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/services/%s/references", serviceName), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to add reference: %s", resp.Status)
	}

	return nil
}

// RemoveServiceReference removes a repository reference from a service
func (c *Client) RemoveServiceReference(ctx context.Context, serviceName, repositoryName string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/services/%s/references/%s", serviceName, repositoryName), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to remove reference: %s", resp.Status)
	}

	return nil
}

// ServiceHealthCheck performs a health check on a service
func (c *Client) ServiceHealthCheck(ctx context.Context, name string) error {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/services/%s/health", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service health check failed: %s", resp.Status)
	}

	return nil
}

// GetService retrieves service information
func (c *Client) GetService(ctx context.Context, name string) (*types.ServiceInstance, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/services/%s", name), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("service not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get service: %s", resp.Status)
	}

	var service types.ServiceInstance
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &service, nil
}
