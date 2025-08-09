package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	WhoopAPIBaseURL = "https://api.prod.whoop.com/developer"
	WhoopAPIVersion = "v2"
)

// WhoopClient handles all interactions with the Whoop API
type WhoopClient struct {
	client       *http.Client
	rateLimiter  *rate.Limiter
	apiKey       string
	refreshToken string
	clientID     string
	clientSecret string
	baseURL      string
}

// NewWhoopClient creates a new Whoop API client with rate limiting
func NewWhoopClient() (*WhoopClient, error) {
	// Try access token first (OAuth), then fall back to API key
	apiKey := os.Getenv("WHOOP_ACCESS_TOKEN")
	if apiKey == "" {
		apiKey = os.Getenv("WHOOP_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("WHOOP_ACCESS_TOKEN or WHOOP_API_KEY environment variable is required")
		}
	}

	// Get refresh token and OAuth credentials for auto-refresh
	refreshToken := os.Getenv("WHOOP_REFRESH_TOKEN")
	clientID := os.Getenv("WHOOP_CLIENT_ID")
	clientSecret := os.Getenv("WHOOP_CLIENT_SECRET")

	// Rate limiter: 100 requests per minute (conservative approach)
	rateLimiter := rate.NewLimiter(rate.Every(time.Minute/100), 10)

	return &WhoopClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter:  rateLimiter,
		apiKey:       apiKey,
		refreshToken: refreshToken,
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      WhoopAPIBaseURL,
	}, nil
}

// makeRequest performs an HTTP request to the Whoop API
func (w *WhoopClient) makeRequest(endpoint string, params url.Values) ([]byte, error) {
	// Wait for rate limiter
	if err := w.rateLimiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	fullURL := w.baseURL + endpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	// Try the request
	body, statusCode, err := w.doRequest(fullURL)
	if err != nil {
		return nil, err
	}

	// If unauthorized and we have refresh capabilities, try to refresh token
	if statusCode == 401 && w.canRefreshToken() {
		log.Printf("Access token expired, attempting to refresh...")

		newToken, err := w.refreshAccessToken()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh access token: %w", err)
		}

		w.apiKey = newToken
		log.Printf("Successfully refreshed access token")

		// Retry the original request with new token
		body, statusCode, err = w.doRequest(fullURL)
		if err != nil {
			return nil, err
		}

		if statusCode == 401 {
			return nil, fmt.Errorf("authentication failed even after token refresh")
		}
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d: %s", statusCode, string(body))
	}

	return body, nil
}

// handleAPIError processes API error responses and returns user-friendly errors
func (w *WhoopClient) handleAPIError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed: invalid API key")
	case http.StatusForbidden:
		return fmt.Errorf("access denied: insufficient permissions")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded: too many requests")
	case http.StatusBadRequest:
		return fmt.Errorf("bad request: check your parameters")
	case http.StatusNotFound:
		return fmt.Errorf("resource not found")
	case http.StatusInternalServerError:
		return fmt.Errorf("Whoop API internal error")
	case http.StatusServiceUnavailable:
		return fmt.Errorf("Whoop API temporarily unavailable")
	default:
		return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
	}
}

// GetUser retrieves the authenticated user's profile information
func (w *WhoopClient) GetUser() (*WhoopUser, error) {
	body, err := w.makeRequest("/v2/user/profile/basic", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	var user WhoopUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user profile: %w", err)
	}

	return &user, nil
}

// GetRecoveryData retrieves recovery data for a date range
func (w *WhoopClient) GetRecoveryData(startDate, endDate time.Time, userID *int) ([]WhoopRecovery, error) {
	params := url.Values{}
	params.Set("start", startDate.Format(time.RFC3339))
	params.Set("end", endDate.Format(time.RFC3339))
	params.Set("limit", "25") // Maximum per request

	var allRecoveries []WhoopRecovery
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/v2/recovery", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get recovery data: %w", err)
		}

		var response WhoopRecoveryResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse recovery data: %w", err)
		}

		allRecoveries = append(allRecoveries, response.Data...)

		// Check if there are more pages
		if response.NextToken == nil || *response.NextToken == "" {
			break
		}
		nextToken = *response.NextToken
	}

	return allRecoveries, nil
}

// GetSleepData retrieves sleep data for a date range
func (w *WhoopClient) GetSleepData(startDate, endDate time.Time, userID *int) ([]WhoopSleep, error) {
	params := url.Values{}
	params.Set("start", startDate.Format(time.RFC3339))
	params.Set("end", endDate.Format(time.RFC3339))
	params.Set("limit", "25") // Maximum per request

	var allSleeps []WhoopSleep
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/v2/activity/sleep", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get sleep data: %w", err)
		}

		var response WhoopSleepResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse sleep data: %w", err)
		}

		allSleeps = append(allSleeps, response.Data...)

		// Check if there are more pages
		if response.NextToken == nil || *response.NextToken == "" {
			break
		}
		nextToken = *response.NextToken
	}

	return allSleeps, nil
}

// GetWorkoutData retrieves workout data for a date range
func (w *WhoopClient) GetWorkoutData(startDate, endDate time.Time, userID *int) ([]WhoopWorkout, error) {
	params := url.Values{}
	params.Set("start", startDate.Format(time.RFC3339))
	params.Set("end", endDate.Format(time.RFC3339))
	params.Set("limit", "25") // Maximum per request

	var allWorkouts []WhoopWorkout
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/v2/activity/workout", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get workout data: %w", err)
		}

		var response WhoopWorkoutResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse workout data: %w", err)
		}

		allWorkouts = append(allWorkouts, response.Data...)

		// Check if there are more pages
		if response.NextToken == nil || *response.NextToken == "" {
			break
		}
		nextToken = *response.NextToken
	}

	return allWorkouts, nil
}

// GetCycleData retrieves physiological cycle data for a date range
func (w *WhoopClient) GetCycleData(startDate, endDate time.Time, userID *int) ([]WhoopCycle, error) {
	params := url.Values{}
	params.Set("start", startDate.Format(time.RFC3339))
	params.Set("end", endDate.Format(time.RFC3339))
	params.Set("limit", "25") // Maximum per request

	var allCycles []WhoopCycle
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/v2/cycle", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get cycle data: %w", err)
		}

		var response WhoopCycleResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse cycle data: %w", err)
		}

		allCycles = append(allCycles, response.Data...)

		// Check if there are more pages
		if response.NextToken == nil || *response.NextToken == "" {
			break
		}
		nextToken = *response.NextToken
	}

	return allCycles, nil
}

// doRequest performs the actual HTTP request
func (w *WhoopClient) doRequest(fullURL string) ([]byte, int, error) {
	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Whoop-MCP-Server/1.0")

	// Execute request
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}

// canRefreshToken checks if we have the necessary credentials for token refresh
func (w *WhoopClient) canRefreshToken() bool {
	return w.refreshToken != "" && w.clientID != "" && w.clientSecret != ""
}

// refreshAccessToken uses the refresh token to get a new access token
func (w *WhoopClient) refreshAccessToken() (string, error) {
	tokenURL := "https://api.prod.whoop.com/oauth/oauth2/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", w.refreshToken)
	data.Set("client_id", w.clientID)
	data.Set("client_secret", w.clientSecret)
	data.Set("scope", "offline")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Update refresh token if a new one was provided
	if tokenResp.RefreshToken != "" {
		w.refreshToken = tokenResp.RefreshToken
	}

	// Optionally update .env file with new tokens
	w.updateEnvFile(tokenResp.AccessToken, w.refreshToken)

	return tokenResp.AccessToken, nil
}

// updateEnvFile updates the .env file with new tokens (optional convenience)
func (w *WhoopClient) updateEnvFile(accessToken, refreshToken string) {
	// This is a best-effort attempt - don't fail if we can't update the file
	envContent := fmt.Sprintf(`# Whoop MCP Server Configuration (V2 API)

# Required: Your Whoop API access token
WHOOP_API_KEY=%s

# Optional: Refresh token for token renewal
WHOOP_REFRESH_TOKEN=%s

# Optional: OAuth credentials for auto-refresh
# WHOOP_CLIENT_ID=your_client_id
# WHOOP_CLIENT_SECRET=your_client_secret

# Optional: Custom API base URL (defaults to production V2)
# WHOOP_API_BASE_URL=https://api.prod.whoop.com/developer

# Optional: Rate limiting configuration (requests per minute)
# WHOOP_RATE_LIMIT=100

# Optional: Request timeout in seconds
# WHOOP_REQUEST_TIMEOUT=30
`, accessToken, refreshToken)

	err := os.WriteFile(".env", []byte(envContent), 0600)
	if err != nil {
		log.Printf("Warning: Could not update .env file with new tokens: %v", err)
	} else {
		log.Printf("Updated .env file with refreshed tokens")
	}
}

// ValidateConnection tests the API connection and authentication
func (w *WhoopClient) ValidateConnection() error {
	_, err := w.GetUser()
	if err != nil {
		return fmt.Errorf("API connection validation failed: %w", err)
	}
	return nil
}
