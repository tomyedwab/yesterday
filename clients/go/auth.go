package yesterdaygo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AccessTokenResponse represents the access token response
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

// Login authenticates the user with username and password
func (c *Client) Login(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return NewValidationError("username and password are required")
	}

	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return NewErrorWithCause(ErrorTypeValidation, "failed to marshal login request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/public/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return NewErrorWithCause(ErrorTypeNetwork, "failed to create login request", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.GetHTTPClient().Do(req)
	if err != nil {
		return NewNetworkError("login request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WrapHTTPError(resp, "login failed")
	}

	// Extract refresh token from YRT cookie
	var refreshToken string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "YRT" {
			refreshToken = cookie.Value
			break
		}
	}

	if refreshToken == "" {
		return NewAuthenticationError("no refresh token received from server")
	}

	// Store refresh token
	if err := c.storeRefreshToken(refreshToken); err != nil {
		return NewErrorWithCause(ErrorTypeNetwork, "failed to store refresh token", err)
	}

	// Try to get access token immediately
	if err := c.RefreshAccessToken(ctx); err != nil {
		// Don't fail login if access token refresh fails - we have the refresh token stored
		// The next API call will trigger another refresh attempt
	}

	return nil
}

// Logout terminates the current session
func (c *Client) Logout(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/public/logout", nil)
	if err != nil {
		return NewErrorWithCause(ErrorTypeNetwork, "failed to create logout request", err)
	}

	// Add refresh token as cookie if we have it
	if refreshToken, err := c.loadRefreshToken(); err == nil && refreshToken != "" {
		req.AddCookie(&http.Cookie{
			Name:  "YRT",
			Value: refreshToken,
		})
	}

	resp, err := c.GetHTTPClient().Do(req)
	if err != nil {
		return NewNetworkError("logout request failed", err)
	}
	defer resp.Body.Close()

	// Clear stored tokens regardless of response status
	c.clearAccessToken()
	c.clearRefreshToken()

	if resp.StatusCode != http.StatusOK {
		return WrapHTTPError(resp, "logout failed")
	}

	return nil
}

// RefreshAccessToken refreshes the access token using the stored refresh token
func (c *Client) RefreshAccessToken(ctx context.Context) error {
	refreshToken, err := c.loadRefreshToken()
	if err != nil {
		return NewErrorWithCause(ErrorTypeAuthentication, "failed to load refresh token", err)
	}

	if refreshToken == "" {
		return NewAuthenticationError("no refresh token available")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/public/access_token", nil)
	if err != nil {
		return NewErrorWithCause(ErrorTypeNetwork, "failed to create access token request", err)
	}

	// Add refresh token as YRT cookie
	req.AddCookie(&http.Cookie{
		Name:  "YRT",
		Value: refreshToken,
	})

	resp, err := c.GetHTTPClient().Do(req)
	if err != nil {
		return NewNetworkError("access token request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WrapHTTPError(resp, "failed to refresh access token")
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewErrorWithCause(ErrorTypeNetwork, "failed to read access token response", err)
	}

	var tokenResp AccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return NewErrorWithCause(ErrorTypeAPI, "failed to parse access token response", err)
	}

	if tokenResp.Error != "" {
		return NewAuthenticationError(tokenResp.Error)
	}

	if tokenResp.AccessToken == "" {
		return NewAuthenticationError("empty access token received")
	}

	// Extract refresh token from YRT cookie
	var newRefreshToken string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "YRT" {
			newRefreshToken = cookie.Value
			break
		}
	}

	if newRefreshToken == "" {
		return NewAuthenticationError("no refresh token received")
	}

	// Store access token in memory
	c.setAccessToken(tokenResp.AccessToken)
	c.storeRefreshToken(newRefreshToken)

	return nil
}

// IsAuthenticated checks if the client has a valid access token
func (c *Client) IsAuthenticated() bool {
	return c.getAccessToken() != ""
}

// storeRefreshToken stores the refresh token to the configured path
func (c *Client) storeRefreshToken(token string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(c.refreshTokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Write token to file with restricted permissions
	if err := os.WriteFile(c.refreshTokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write refresh token: %w", err)
	}

	return nil
}

// loadRefreshToken loads the refresh token from the configured path
func (c *Client) loadRefreshToken() (string, error) {
	data, err := os.ReadFile(c.refreshTokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No token file is not an error
		}
		return "", fmt.Errorf("failed to read refresh token: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// clearRefreshToken removes the stored refresh token
func (c *Client) clearRefreshToken() {
	os.Remove(c.refreshTokenPath)
}
