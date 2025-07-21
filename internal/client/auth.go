package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token   string `json:"token"`
	User    User   `json:"user"`
	Message string `json:"message,omitempty"`
}

// User represents user information
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}

// Login authenticates with the server
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	req := LoginRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/auth/login", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("login failed: %s", resp.Status)
		}
		return nil, fmt.Errorf("login failed: %s", errResp.Error)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Save the token
	if err := c.SetToken(loginResp.Token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return &loginResp, nil
}

// Logout logs out from the server
func (c *Client) Logout(ctx context.Context) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("not authenticated")
	}

	resp, err := c.doRequest(ctx, "POST", "/api/auth/logout", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Clear token regardless of server response
	if err := c.ClearToken(); err != nil {
		return fmt.Errorf("failed to clear token: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("logout failed: %s", resp.Status)
	}

	return nil
}

// WhoAmI returns information about the current user
func (c *Client) WhoAmI(ctx context.Context) (*User, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	resp, err := c.doRequest(ctx, "GET", "/api/auth/whoami", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: %s", resp.Status)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}
