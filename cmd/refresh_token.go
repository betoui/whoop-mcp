package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: go run cmd/refresh_token.go <client_id> <client_secret> <refresh_token>")
		fmt.Println("")
		fmt.Println("This will use your refresh token to get a new access token.")
		return
	}

	clientID := os.Args[1]
	clientSecret := os.Args[2]
	refreshToken := os.Args[3]

	fmt.Println("🔄 Refreshing access token...")

	// Prepare the token refresh request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequest("POST", "https://api.prod.whoop.com/oauth/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error making token refresh request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Token refresh failed (status %d):\n%s\n", resp.StatusCode, string(body))
		fmt.Println("")
		fmt.Println("Common issues:")
		fmt.Println("- Refresh token expired (they last much longer but do expire)")
		fmt.Println("- Invalid client credentials")
		fmt.Println("- Refresh token already used (some implementations are single-use)")
		return
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		fmt.Printf("❌ Error parsing token response: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully refreshed tokens!")
	fmt.Println("")
	fmt.Println("📝 Your new tokens:")
	fmt.Printf("Access Token:  %s\n", tokenResp.AccessToken)
	if tokenResp.RefreshToken != "" {
		fmt.Printf("Refresh Token: %s\n", tokenResp.RefreshToken)
	}
	fmt.Printf("Expires in:    %d seconds (%.1f hours)\n", tokenResp.ExpiresIn, float64(tokenResp.ExpiresIn)/3600)
	fmt.Printf("Scopes:        %s\n", tokenResp.Scope)

	// Write to .env file
	writeEnvFile(tokenResp.AccessToken, tokenResp.RefreshToken)
}

func writeEnvFile(accessToken, refreshToken string) {
	envContent := fmt.Sprintf(`# Whoop MCP Server Configuration (V2 API)

# Required: Your Whoop API access token
WHOOP_API_KEY=%s

# Optional: Refresh token for token renewal
WHOOP_REFRESH_TOKEN=%s

# Optional: Custom API base URL (defaults to production V2)
# WHOOP_API_BASE_URL=https://api.prod.whoop.com/developer

# Optional: Rate limiting configuration (requests per minute)
# WHOOP_RATE_LIMIT=100

# Optional: Request timeout in seconds
# WHOOP_REQUEST_TIMEOUT=30

# Optional: Enable debug logging
# DEBUG=false
`, accessToken, refreshToken)

	err := os.WriteFile(".env", []byte(envContent), 0600)
	if err != nil {
		fmt.Printf("⚠️  Could not write .env file: %v\n", err)
		fmt.Println("Please create .env manually with the token above.")
	} else {
		fmt.Println("✅ Updated .env file with your new tokens!")
		fmt.Println("")
		fmt.Println("🚀 Your MCP server is now ready to use!")
	}
}
