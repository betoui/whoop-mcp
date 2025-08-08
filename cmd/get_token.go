package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Whoop OAuth Token Helper")
		fmt.Println("========================")
		fmt.Println("")
		fmt.Println("Usage: go run cmd/get_token.go <client_id> <client_secret> [authorization_code]")
		fmt.Println("")
		fmt.Println("Step 1: Get authorization URL")
		fmt.Println("  go run cmd/get_token.go <client_id> <client_secret>")
		fmt.Println("")
		fmt.Println("Step 2: Exchange code for token")
		fmt.Println("  go run cmd/get_token.go <client_id> <client_secret> <auth_code>")
		return
	}

	clientID := os.Args[1]
	clientSecret := os.Args[2]

	if len(os.Args) == 3 {
		// Step 1: Generate authorization URL
		generateAuthURL(clientID)
	} else {
		// Step 2: Exchange authorization code for token
		authCode := os.Args[3]
		exchangeCodeForToken(clientID, clientSecret, authCode)
	}
}

func generateAuthURL(clientID string) {
	baseURL := "https://api.prod.whoop.com/oauth/oauth2/auth"

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", "http://localhost:3000/callback") // Use your registered redirect URI
	params.Set("response_type", "code")
	params.Set("scope", "read:recovery read:sleep read:workout read:cycles read:profile offline")
	params.Set("state", "whoop-mcp-auth") // 8+ character state for security

	authURL := baseURL + "?" + params.Encode()

	fmt.Println("🔗 STEP 1: Open this URL in your browser to authorize the app:")
	fmt.Println("")
	fmt.Println(authURL)
	fmt.Println("")
	fmt.Println("After authorizing, you'll be redirected to a URL like:")
	fmt.Println("http://localhost:3000/callback?code=AUTHORIZATION_CODE&state=whoop-mcp-auth")
	fmt.Println("")
	fmt.Println("📋 STEP 2: Copy the 'code' parameter and run:")
	fmt.Printf("go run cmd/get_token.go %s [your_client_secret] <AUTHORIZATION_CODE>\n", clientID)
	fmt.Println("")
	fmt.Println("⚠️  Note: The redirect URL might show an error page, that's OK!")
	fmt.Println("   Just copy the 'code' parameter from the URL bar.")
}

func exchangeCodeForToken(clientID, clientSecret, authCode string) {
	fmt.Println("🔄 Exchanging authorization code for access token...")

	tokenURL := "https://api.prod.whoop.com/oauth/oauth2/token"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", "http://localhost:3000/callback")
	data.Set("code", authCode)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		fmt.Printf("❌ Error making token request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Token request failed (status %d):\n%s\n", resp.StatusCode, string(body))
		fmt.Println("")
		fmt.Println("Common issues:")
		fmt.Println("- Authorization code already used (codes are single-use)")
		fmt.Println("- Authorization code expired (they expire quickly)")
		fmt.Println("- Wrong redirect URI (must match exactly)")
		fmt.Println("- Invalid client credentials")
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

	fmt.Println("✅ Successfully obtained tokens!")
	fmt.Println("")
	fmt.Println("📝 Your tokens:")
	fmt.Printf("Access Token:  %s\n", tokenResp.AccessToken)
	if tokenResp.RefreshToken != "" {
		fmt.Printf("Refresh Token: %s\n", tokenResp.RefreshToken)
	}
	fmt.Printf("Expires in:    %d seconds (%d hours)\n", tokenResp.ExpiresIn, tokenResp.ExpiresIn/3600)
	fmt.Printf("Scopes:        %s\n", tokenResp.Scope)
	fmt.Println("")

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
		fmt.Println("✅ Created .env file with your tokens!")
		fmt.Println("")
		fmt.Println("🚀 Next steps:")
		fmt.Println("1. Build the MCP server: make build")
		fmt.Println("2. Test the server: ./bin/whoop-mcp-server")
		fmt.Println("3. Configure Claude Desktop (see README)")
	}
}
