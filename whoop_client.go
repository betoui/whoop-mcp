package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/time/rate"
)

const (
	WhoopAPIBaseURL = "https://api.prod.whoop.com/developer"
	WhoopAPIVersion = "v2"
)

// WhoopClient handles all interactions with the Whoop API
type WhoopClient struct {
	client      *http.Client
	rateLimiter *rate.Limiter
	apiKey      string
	baseURL     string
}

// NewWhoopClient creates a new Whoop API client with rate limiting
func NewWhoopClient() (*WhoopClient, error) {
	apiKey := os.Getenv("WHOOP_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("WHOOP_API_KEY environment variable is required")
	}

	// Rate limiter: 100 requests per minute (conservative approach)
	rateLimiter := rate.NewLimiter(rate.Every(time.Minute/100), 10)

	return &WhoopClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: rateLimiter,
		apiKey:      apiKey,
		baseURL:     WhoopAPIBaseURL + "/" + WhoopAPIVersion,
	}, nil
}

// makeRequest performs an authenticated HTTP request to the Whoop API
func (w *WhoopClient) makeRequest(endpoint string, params url.Values) ([]byte, error) {
	// Respect rate limiting
	if err := w.rateLimiter.Wait(nil); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build URL
	requestURL := w.baseURL + endpoint
	if params != nil && len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	// Create request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Whoop-MCP-Server/1.0")

	// Execute request
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, w.handleAPIError(resp.StatusCode, body)
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
	body, err := w.makeRequest("/user/profile/basic", nil)
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
	params.Set("start", startDate.Format("2006-01-02T15:04:05Z"))
	params.Set("end", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("limit", "50") // Maximum per request

	var allRecoveries []WhoopRecovery
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/recovery", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get recovery data: %w", err)
		}

		var response WhoopRecoveryResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse recovery data: %w", err)
		}

		allRecoveries = append(allRecoveries, response.Data...)

		// Check if there's more data
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
	params.Set("start", startDate.Format("2006-01-02T15:04:05Z"))
	params.Set("end", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("limit", "50")

	var allSleep []WhoopSleep
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/activity/sleep", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get sleep data: %w", err)
		}

		var response WhoopSleepResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse sleep data: %w", err)
		}

		allSleep = append(allSleep, response.Data...)

		if response.NextToken == nil || *response.NextToken == "" {
			break
		}
		nextToken = *response.NextToken
	}

	return allSleep, nil
}

// GetWorkoutData retrieves workout data for a date range
func (w *WhoopClient) GetWorkoutData(startDate, endDate time.Time, userID *int) ([]WhoopWorkout, error) {
	params := url.Values{}
	params.Set("start", startDate.Format("2006-01-02T15:04:05Z"))
	params.Set("end", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("limit", "50")

	var allWorkouts []WhoopWorkout
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/activity/workout", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get workout data: %w", err)
		}

		var response WhoopWorkoutResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse workout data: %w", err)
		}

		allWorkouts = append(allWorkouts, response.Data...)

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
	params.Set("start", startDate.Format("2006-01-02T15:04:05Z"))
	params.Set("end", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("limit", "50")

	var allCycles []WhoopCycle
	nextToken := ""

	for {
		if nextToken != "" {
			params.Set("nextToken", nextToken)
		}

		body, err := w.makeRequest("/cycle", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get cycle data: %w", err)
		}

		var response WhoopCycleResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse cycle data: %w", err)
		}

		allCycles = append(allCycles, response.Data...)

		if response.NextToken == nil || *response.NextToken == "" {
			break
		}
		nextToken = *response.NextToken
	}

	return allCycles, nil
}

// ValidateConnection tests the API connection and authentication
func (w *WhoopClient) ValidateConnection() error {
	_, err := w.GetUser()
	if err != nil {
		return fmt.Errorf("API connection validation failed: %w", err)
	}
	return nil
}
