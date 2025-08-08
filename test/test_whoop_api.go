package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	accessToken := os.Getenv("WHOOP_API_KEY")
	if accessToken == "" {
		fmt.Println("❌ WHOOP_API_KEY not found in environment")
		return
	}

	fmt.Printf("🔗 Testing Whoop API endpoints with token: %s...\n", accessToken[:20])
	fmt.Println()

	// Test 1: User Profile
	fmt.Println("1️⃣ Testing User Profile...")
	testEndpoint("https://api.prod.whoop.com/developer/v2/user/profile/basic", accessToken, nil)

	// Test 2: Recent Recovery (last 7 days)
	fmt.Println("\n2️⃣ Testing Recovery Data...")
	params := url.Values{}
	params.Set("limit", "5")
	end := time.Now()
	start := end.AddDate(0, 0, -7) // Last 7 days
	params.Set("start", start.Format(time.RFC3339))
	params.Set("end", end.Format(time.RFC3339))
	testEndpoint("https://api.prod.whoop.com/developer/v2/recovery", accessToken, params)

	// Test 3: Recent Sleep (last 7 days)
	fmt.Println("\n3️⃣ Testing Sleep Data...")
	testEndpoint("https://api.prod.whoop.com/developer/v2/activity/sleep", accessToken, params)

	// Test 4: Recent Workouts (last 7 days)
	fmt.Println("\n4️⃣ Testing Workout Data...")
	testEndpoint("https://api.prod.whoop.com/developer/v2/activity/workout", accessToken, params)

	// Test 5: Recent Cycles (last 7 days)
	fmt.Println("\n5️⃣ Testing Cycle Data...")
	testEndpoint("https://api.prod.whoop.com/developer/v2/cycle", accessToken, params)
}

func testEndpoint(baseURL string, accessToken string, params url.Values) {
	// Build URL with parameters
	requestURL := baseURL
	if params != nil && len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	fmt.Printf("   📡 GET %s\n", requestURL)

	// Create request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		fmt.Printf("   ❌ Failed to create request: %v\n", err)
		return
	}

	// Add auth header
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   ❌ Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("   ❌ Failed to read response: %v\n", err)
		return
	}

	// Check status
	if resp.StatusCode != 200 {
		fmt.Printf("   ❌ HTTP %d: %s\n", resp.StatusCode, string(body))
		return
	}

	// Parse and display JSON
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("   ❌ Failed to parse JSON: %v\n", err)
		fmt.Printf("   Raw response: %s\n", string(body))
		return
	}

	// Pretty print JSON
	prettyJSON, err := json.MarshalIndent(result, "   ", "  ")
	if err != nil {
		fmt.Printf("   ❌ Failed to format JSON: %v\n", err)
		return
	}

	fmt.Printf("   ✅ Success (%d bytes):\n", len(body))
	fmt.Printf("%s\n", string(prettyJSON))
}
