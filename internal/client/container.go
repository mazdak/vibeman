package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"vibeman/internal/container"
)

// CreateContainerRequest represents the container creation request
type CreateContainerRequest struct {
	RepositoryName string `json:"projectName"`
	Environment string `json:"environment"`
	Image       string `json:"image"`
}

// Container operations

// CreateContainer creates a new container
func (c *Client) CreateContainer(ctx context.Context, repositoryName, environment, image string) (*container.Container, error) {
	req := CreateContainerRequest{
		RepositoryName: repositoryName,
		Environment: environment,
		Image:       image,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/containers", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create container: %s", resp.Status)
	}

	var cont container.Container
	if err := json.NewDecoder(resp.Body).Decode(&cont); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &cont, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/containers/%s/start", containerID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to start container: %s", resp.Status)
	}

	return nil
}

// StopContainer stops a container
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/containers/%s/stop", containerID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to stop container: %s", resp.Status)
	}

	return nil
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/containers/%s", containerID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to remove container: %s", resp.Status)
	}

	return nil
}

// ListContainers lists all containers
func (c *Client) ListContainers(ctx context.Context) ([]*container.Container, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/containers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list containers: %s", resp.Status)
	}

	var containers []*container.Container
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return containers, nil
}

// GetContainerByName gets a container by name
func (c *Client) GetContainerByName(ctx context.Context, name string) (*container.Container, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/containers/name/%s", name), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("container not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get container: %s", resp.Status)
	}

	var cont container.Container
	if err := json.NewDecoder(resp.Body).Decode(&cont); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &cont, nil
}

// GetContainersByRepository gets containers by project
func (c *Client) GetContainersByRepository(ctx context.Context, project string) ([]*container.Container, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/containers/project/%s", project), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get containers: %s", resp.Status)
	}

	var containers []*container.Container
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return containers, nil
}

// ExecContainer executes a command in a container
func (c *Client) ExecContainer(ctx context.Context, containerID string, command []string) ([]byte, error) {
	req := struct {
		Command []string `json:"command"`
	}{
		Command: command,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/containers/%s/exec", containerID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to exec in container: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// ContainerLogs retrieves container logs
func (c *Client) ContainerLogs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	path := fmt.Sprintf("/api/containers/%s/logs", containerID)
	if follow {
		path += "?follow=true"
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get logs: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// ContainerShell opens an interactive shell in a container (WebSocket)
func (c *Client) ContainerShell(ctx context.Context, containerID string, shell string) (*websocket.Conn, error) {
	path := fmt.Sprintf("/api/containers/%s/shell?shell=%s", containerID, shell)
	return c.WebSocketConnect(ctx, path)
}

// ContainerAttach attaches to a container (WebSocket)
func (c *Client) ContainerAttach(ctx context.Context, containerID string) (*websocket.Conn, error) {
	path := fmt.Sprintf("/api/containers/%s/attach", containerID)
	return c.WebSocketConnect(ctx, path)
}

// CopyToContainer copies files to a container
func (c *Client) CopyToContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	req := struct {
		SrcPath string `json:"srcPath"`
		DstPath string `json:"dstPath"`
	}{
		SrcPath: srcPath,
		DstPath: dstPath,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/containers/%s/copy/to", containerID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to copy to container: %s", resp.Status)
	}

	return nil
}

// CopyFromContainer copies files from a container
func (c *Client) CopyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	req := struct {
		SrcPath string `json:"srcPath"`
		DstPath string `json:"dstPath"`
	}{
		SrcPath: srcPath,
		DstPath: dstPath,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/containers/%s/copy/from", containerID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to copy from container: %s", resp.Status)
	}

	return nil
}

// RunContainerSetup runs setup in a container
func (c *Client) RunContainerSetup(ctx context.Context, containerID string, projectPath string) error {
	req := struct {
		ProjectPath string `json:"projectPath"`
	}{
		ProjectPath: projectPath,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/containers/%s/setup", containerID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to run setup: %s", resp.Status)
	}

	return nil
}

// RunContainerLifecycleHook runs a lifecycle hook in a container
func (c *Client) RunContainerLifecycleHook(ctx context.Context, containerID string, hook string) error {
	req := struct {
		Hook string `json:"hook"`
	}{
		Hook: hook,
	}

	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/containers/%s/lifecycle", containerID), req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to run lifecycle hook: %s", resp.Status)
	}

	return nil
}
