package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// MCPServer handles the Model Context Protocol communication
type MCPServer struct {
	whoopClient    *WhoopClient
	healthAnalyzer *HealthAnalyzer
	tools          []MCPTool
	resources      []MCPResource
	initialized    bool
	mu             sync.RWMutex
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer() (*MCPServer, error) {
	whoopClient, err := NewWhoopClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Whoop client: %w", err)
	}

	healthAnalyzer := NewHealthAnalyzer()

	server := &MCPServer{
		whoopClient:    whoopClient,
		healthAnalyzer: healthAnalyzer,
		tools:          defineMCPTools(),
		resources:      defineMCPResources(),
		initialized:    false,
	}

	return server, nil
}

// Run starts the MCP server and handles stdio communication
func (s *MCPServer) Run() error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse the incoming JSON-RPC message
		var request MCPRequest
		if err := json.Unmarshal(line, &request); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		// Handle the request
		s.handleRequest(&request)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	return nil
}

// handleRequest processes incoming MCP requests
func (s *MCPServer) handleRequest(request *MCPRequest) {
	switch request.Method {
	case "initialize":
		s.handleInitialize(request)
	case "tools/list":
		s.handleToolsList(request)
	case "tools/call":
		s.handleToolsCall(request)
	case "resources/list":
		s.handleResourcesList(request)
	case "resources/read":
		s.handleResourcesRead(request)
	default:
		s.sendError(request.ID, -32601, "Method not found", fmt.Sprintf("Unknown method: %s", request.Method))
	}
}

// handleInitialize processes the initialize request
func (s *MCPServer) handleInitialize(request *MCPRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate API connection
	if err := s.whoopClient.ValidateConnection(); err != nil {
		s.sendError(request.ID, -32603, "Internal error", fmt.Sprintf("Failed to connect to Whoop API: %v", err))
		return
	}

	s.initialized = true

	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "whoop-mcp-server",
			"version": "1.0.0",
		},
	}

	s.sendResponse(request.ID, result)
}

// handleToolsList returns the list of available tools
func (s *MCPServer) handleToolsList(request *MCPRequest) {
	if !s.isInitialized() {
		s.sendError(request.ID, -32002, "Not initialized", "Server not initialized")
		return
	}

	result := map[string]interface{}{
		"tools": s.tools,
	}

	s.sendResponse(request.ID, result)
}

// handleToolsCall executes a tool call
func (s *MCPServer) handleToolsCall(request *MCPRequest) {
	if !s.isInitialized() {
		s.sendError(request.ID, -32002, "Not initialized", "Server not initialized")
		return
	}

	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		s.sendError(request.ID, -32602, "Invalid params", err.Error())
		return
	}

	// Execute the tool
	result, err := s.executeTool(params.Name, params.Arguments)
	if err != nil {
		s.sendError(request.ID, -32603, "Internal error", err.Error())
		return
	}

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result,
			},
		},
	}

	s.sendResponse(request.ID, response)
}

// handleResourcesList returns the list of available resources
func (s *MCPServer) handleResourcesList(request *MCPRequest) {
	if !s.isInitialized() {
		s.sendError(request.ID, -32002, "Not initialized", "Server not initialized")
		return
	}

	result := map[string]interface{}{
		"resources": s.resources,
	}

	s.sendResponse(request.ID, result)
}

// handleResourcesRead reads a specific resource
func (s *MCPServer) handleResourcesRead(request *MCPRequest) {
	if !s.isInitialized() {
		s.sendError(request.ID, -32002, "Not initialized", "Server not initialized")
		return
	}

	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		s.sendError(request.ID, -32602, "Invalid params", err.Error())
		return
	}

	// Read the resource
	content, err := s.readResource(params.URI)
	if err != nil {
		s.sendError(request.ID, -32603, "Internal error", err.Error())
		return
	}

	result := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      params.URI,
				"mimeType": "application/json",
				"text":     content,
			},
		},
	}

	s.sendResponse(request.ID, result)
}

// sendResponse sends a successful JSON-RPC response
func (s *MCPServer) sendResponse(id interface{}, result interface{}) {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	s.writeMessage(response)
}

// sendError sends an error JSON-RPC response
func (s *MCPServer) sendError(id interface{}, code int, message string, data interface{}) {
	// Don't send error responses for notifications (null or missing ID)
	if id == nil {
		log.Printf("Error for notification (no response sent): %s - %v", message, data)
		return
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	s.writeMessage(response)
}

// writeMessage writes a message to stdout
func (s *MCPServer) writeMessage(message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	_, err = fmt.Fprintf(os.Stdout, "%s\n", data)
	if err != nil {
		log.Printf("Error writing message: %v", err)
	}
}

// isInitialized checks if the server is initialized
func (s *MCPServer) isInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// defineMCPTools defines the available MCP tools
func defineMCPTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "get_health_summary",
			Description: "Get a comprehensive health summary for therapy sessions including recovery trends, sleep analysis, stress indicators, and actionable insights",
			InputSchema: MCPInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"start_date": map[string]interface{}{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"end_date": map[string]interface{}{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"user_id": map[string]interface{}{
						"type":        "integer",
						"description": "Optional user ID (defaults to authenticated user)",
					},
				},
				Required: []string{"start_date", "end_date"},
			},
		},
		{
			Name:        "analyze_stress_indicators",
			Description: "Analyze physiological stress markers from HRV, resting heart rate, and recovery patterns to identify mental health concerns",
			InputSchema: MCPInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"start_date": map[string]interface{}{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"end_date": map[string]interface{}{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"user_id": map[string]interface{}{
						"type":        "integer",
						"description": "Optional user ID (defaults to authenticated user)",
					},
				},
				Required: []string{"start_date", "end_date"},
			},
		},
		{
			Name:        "analyze_sleep_patterns",
			Description: "Analyze sleep quality, patterns, and their impact on mental health for therapeutic conversations",
			InputSchema: MCPInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"start_date": map[string]interface{}{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"end_date": map[string]interface{}{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"user_id": map[string]interface{}{
						"type":        "integer",
						"description": "Optional user ID (defaults to authenticated user)",
					},
				},
				Required: []string{"start_date", "end_date"},
			},
		},
		{
			Name:        "analyze_activity_patterns",
			Description: "Analyze workout patterns, exercise habits, and their relationship to mental health and behavioral insights",
			InputSchema: MCPInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"start_date": map[string]interface{}{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"end_date": map[string]interface{}{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format",
						"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
					},
					"user_id": map[string]interface{}{
						"type":        "integer",
						"description": "Optional user ID (defaults to authenticated user)",
					},
				},
				Required: []string{"start_date", "end_date"},
			},
		},
		{
			Name:        "analyze_health_trends",
			Description: "Analyze week-over-week trends in recovery, sleep, or strain metrics to identify patterns relevant for therapy",
			InputSchema: MCPInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"metric": map[string]interface{}{
						"type":        "string",
						"description": "Metric to analyze: recovery, sleep, or strain",
						"enum":        []string{"recovery", "sleep", "strain"},
					},
					"days": map[string]interface{}{
						"type":        "integer",
						"description": "Number of days to analyze (default: 14)",
						"minimum":     7,
						"maximum":     90,
					},
					"user_id": map[string]interface{}{
						"type":        "integer",
						"description": "Optional user ID (defaults to authenticated user)",
					},
				},
				Required: []string{"metric"},
			},
		},
		{
			Name:        "setup_whoop_auth",
			Description: "Guide user through Whoop OAuth setup process",
			InputSchema: MCPInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"client_id": map[string]interface{}{
						"type":        "string",
						"description": "Whoop app client ID (optional, will generate URL if provided)",
					},
					"authorization_code": map[string]interface{}{
						"type":        "string",
						"description": "Authorization code from Whoop (optional, for token exchange)",
					},
					"client_secret": map[string]interface{}{
						"type":        "string",
						"description": "Whoop app client secret (required if authorization_code provided)",
					},
				},
			},
		},
	}
}

// defineMCPResources defines the available MCP resources
func defineMCPResources() []MCPResource {
	return []MCPResource{
		{
			URI:         "whoop://user/profile",
			Name:        "User Profile",
			Description: "Basic user profile information",
			MimeType:    "application/json",
		},
		{
			URI:         "whoop://health/recent",
			Name:        "Recent Health Data",
			Description: "Most recent recovery, sleep, and activity data",
			MimeType:    "application/json",
		},
	}
}

// executeTool executes a specific tool with the given arguments
func (s *MCPServer) executeTool(toolName string, arguments json.RawMessage) (string, error) {
	switch toolName {
	case "get_health_summary":
		return s.executeHealthSummaryTool(arguments)
	case "analyze_stress_indicators":
		return s.executeStressAnalysisTool(arguments)
	case "analyze_sleep_patterns":
		return s.executeSleepAnalysisTool(arguments)
	case "analyze_activity_patterns":
		return s.executeActivityAnalysisTool(arguments)
	case "analyze_health_trends":
		return s.executeTrendAnalysisTool(arguments)
	case "setup_whoop_auth":
		return s.executeWhoopAuthSetupTool(arguments)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// parseDateRange parses start and end dates, adjusting for same-day queries
func parseDateRange(startDateStr, endDateStr string) (time.Time, time.Time, error) {
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date format: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date format: %w", err)
	}

	// For same-day queries, extend end date to end of day to capture all data
	// Use AddDate to go to the next day, then subtract 1 second to get 23:59:59
	if startDate.Format("2006-01-02") == endDate.Format("2006-01-02") {
		endDate = endDate.AddDate(0, 0, 1).Add(-time.Second)
	}

	return startDate, endDate, nil
}

// executeHealthSummaryTool implements the health summary tool
func (s *MCPServer) executeHealthSummaryTool(arguments json.RawMessage) (string, error) {
	var input HealthSummaryInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Parse dates
	startDate, endDate, err := parseDateRange(input.StartDate, input.EndDate)
	if err != nil {
		return "", err
	}

	// Validate date range
	if endDate.Before(startDate) {
		return "", fmt.Errorf("end_date must be after start_date")
	}

	// Get user ID
	userID := 0
	if input.UserID != nil {
		userID = *input.UserID
	} else {
		user, err := s.whoopClient.GetUser()
		if err != nil {
			return "", fmt.Errorf("failed to get user: %w", err)
		}
		userID = user.UserID
	}

	// Fetch all health data concurrently
	var recoveries []WhoopRecovery
	var sleepData []WhoopSleep
	var workouts []WhoopWorkout
	var cycles []WhoopCycle

	// Create error channel for concurrent operations
	errCh := make(chan error, 4)
	var wg sync.WaitGroup

	// Fetch recovery data
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.whoopClient.GetRecoveryData(startDate, endDate, &userID)
		if err != nil {
			errCh <- fmt.Errorf("failed to get recovery data: %w", err)
			return
		}
		recoveries = data
	}()

	// Fetch sleep data
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.whoopClient.GetSleepData(startDate, endDate, &userID)
		if err != nil {
			errCh <- fmt.Errorf("failed to get sleep data: %w", err)
			return
		}
		sleepData = data
	}()

	// Fetch workout data
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.whoopClient.GetWorkoutData(startDate, endDate, &userID)
		if err != nil {
			errCh <- fmt.Errorf("failed to get workout data: %w", err)
			return
		}
		workouts = data
	}()

	// Fetch cycle data
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.whoopClient.GetCycleData(startDate, endDate, &userID)
		if err != nil {
			errCh <- fmt.Errorf("failed to get cycle data: %w", err)
			return
		}
		cycles = data
	}()

	// Wait for all operations to complete
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		if err != nil {
			return "", err
		}
	}

	// Analyze the data
	summary, err := s.healthAnalyzer.AnalyzeHealthSummary(recoveries, sleepData, workouts, cycles, startDate, endDate, userID)
	if err != nil {
		return "", fmt.Errorf("failed to analyze health data: %w", err)
	}

	// Format for therapy
	return s.healthAnalyzer.FormatInsightsForTherapy(summary), nil
}

// executeStressAnalysisTool implements the stress analysis tool
func (s *MCPServer) executeStressAnalysisTool(arguments json.RawMessage) (string, error) {
	var input StressAnalysisInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	startDate, endDate, err := parseDateRange(input.StartDate, input.EndDate)
	if err != nil {
		return "", err
	}

	userID := 0
	if input.UserID != nil {
		userID = *input.UserID
	}

	// Get recovery data for stress analysis
	recoveries, err := s.whoopClient.GetRecoveryData(startDate, endDate, &userID)
	if err != nil {
		return "", fmt.Errorf("failed to get recovery data: %w", err)
	}

	sleepData, err := s.whoopClient.GetSleepData(startDate, endDate, &userID)
	if err != nil {
		return "", fmt.Errorf("failed to get sleep data: %w", err)
	}

	// Analyze stress indicators
	stressIndicators := s.healthAnalyzer.analyzeStressIndicators(recoveries, sleepData)

	return fmt.Sprintf(`# Stress Analysis Report

**Analysis Period:** %s to %s

## Physiological Stress Indicators

- **Overall Stress Level:** %s
- **Physiological Stress Score:** %.1f/100
- **Days with Elevated HRV:** %d
- **Days with High Resting HR:** %d
- **Poor Recovery Streak:** %d days

## Interpretation

The physiological stress score combines multiple biomarkers including heart rate variability patterns, resting heart rate elevations, and recovery consistency. 

**Stress Level Definitions:**
- **Low (0-30):** Normal physiological stress response
- **Moderate (30-50):** Elevated stress requiring attention
- **High (50-70):** Significant stress impacting recovery
- **Critical (70+):** Severe stress requiring immediate intervention

## Therapeutic Considerations

%s

*Note: This analysis is based on physiological markers and should be combined with psychological assessment for comprehensive evaluation.*`,
		input.StartDate, input.EndDate,
		stressIndicators.StressLevel,
		stressIndicators.PhysiologicalStress,
		stressIndicators.ElevatedHRVDays,
		stressIndicators.HighRestingHRDays,
		stressIndicators.PoorRecoveryStreak,
		s.getStressRecommendations(stressIndicators)), nil
}

// executeSleepAnalysisTool implements the sleep analysis tool
func (s *MCPServer) executeSleepAnalysisTool(arguments json.RawMessage) (string, error) {
	var input SleepAnalysisInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	startDate, endDate, err := parseDateRange(input.StartDate, input.EndDate)
	if err != nil {
		return "", err
	}

	userID := 0
	if input.UserID != nil {
		userID = *input.UserID
	}

	sleepData, err := s.whoopClient.GetSleepData(startDate, endDate, &userID)
	if err != nil {
		return "", fmt.Errorf("failed to get sleep data: %w", err)
	}

	analysis := s.healthAnalyzer.analyzeSleepPatterns(sleepData)

	return fmt.Sprintf(`# Sleep Pattern Analysis

**Analysis Period:** %s to %s
**Total Sleep Sessions:** %d

## Sleep Metrics

- **Average Duration:** %.1f hours
- **Sleep Efficiency:** %.1f%%
- **Average Sleep Debt:** %.1f hours
- **Sleep Consistency Score:** %.1f%% 
- **Average Disturbances:** %.1f per night
- **Quality Trend:** %s

## Mental Health Implications

%s

## Recommendations

%s`,
		input.StartDate, input.EndDate, len(sleepData),
		analysis.AverageHours,
		analysis.AverageEfficiency*100,
		analysis.AverageDebt,
		analysis.ConsistencyScore*100,
		analysis.DisturbanceFrequency,
		analysis.SleepQualityTrend,
		s.getSleepMentalHealthImplications(analysis),
		s.getSleepRecommendations(analysis)), nil
}

// executeActivityAnalysisTool implements the activity analysis tool
func (s *MCPServer) executeActivityAnalysisTool(arguments json.RawMessage) (string, error) {
	var input SleepAnalysisInput // Reusing same input structure
	if err := json.Unmarshal(arguments, &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	startDate, endDate, err := parseDateRange(input.StartDate, input.EndDate)
	if err != nil {
		return "", err
	}

	userID := 0
	if input.UserID != nil {
		userID = *input.UserID
	}

	workouts, err := s.whoopClient.GetWorkoutData(startDate, endDate, &userID)
	if err != nil {
		return "", fmt.Errorf("failed to get workout data: %w", err)
	}

	cycles, err := s.whoopClient.GetCycleData(startDate, endDate, &userID)
	if err != nil {
		return "", fmt.Errorf("failed to get cycle data: %w", err)
	}

	patterns := s.healthAnalyzer.analyzeActivityPatterns(workouts, cycles)

	return fmt.Sprintf(`# Activity Pattern Analysis

**Analysis Period:** %s to %s
**Total Workouts:** %d

## Activity Metrics

- **Weekly Workout Frequency:** %d sessions
- **Average Strain:** %.1f
- **Workout Consistency:** %.1f%%
- **Overtraining Risk:** %s
- **Active Recovery Days:** %d
- **Intensity Balance:** %s

## Behavioral Health Insights

%s`,
		input.StartDate, input.EndDate, len(workouts),
		patterns.WeeklyWorkouts,
		patterns.AverageStrain,
		patterns.WorkoutConsistency*100,
		patterns.OvertrainingRisk,
		patterns.ActiveRecoveryDays,
		patterns.IntensityBalance,
		s.getActivityBehavioralInsights(patterns)), nil
}

// executeTrendAnalysisTool implements the trend analysis tool
func (s *MCPServer) executeTrendAnalysisTool(arguments json.RawMessage) (string, error) {
	var input TrendAnalysisInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	days := input.Days
	if days == 0 {
		days = 14 // Default to 2 weeks
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	userID := 0
	if input.UserID != nil {
		userID = *input.UserID
	}

	switch input.Metric {
	case "recovery":
		recoveries, err := s.whoopClient.GetRecoveryData(startDate, endDate, &userID)
		if err != nil {
			return "", fmt.Errorf("failed to get recovery data: %w", err)
		}
		trend := s.healthAnalyzer.analyzeRecoveryTrend(recoveries)
		return s.formatRecoveryTrend(trend, days), nil

	case "sleep":
		sleepData, err := s.whoopClient.GetSleepData(startDate, endDate, &userID)
		if err != nil {
			return "", fmt.Errorf("failed to get sleep data: %w", err)
		}
		analysis := s.healthAnalyzer.analyzeSleepPatterns(sleepData)
		return s.formatSleepTrend(analysis, days), nil

	case "strain":
		cycles, err := s.whoopClient.GetCycleData(startDate, endDate, &userID)
		if err != nil {
			return "", fmt.Errorf("failed to get cycle data: %w", err)
		}
		return s.formatStrainTrend(cycles, days), nil

	default:
		return "", fmt.Errorf("unsupported metric: %s", input.Metric)
	}
}

// readResource reads a specific resource
func (s *MCPServer) readResource(uri string) (string, error) {
	switch uri {
	case "whoop://user/profile":
		user, err := s.whoopClient.GetUser()
		if err != nil {
			return "", fmt.Errorf("failed to get user profile: %w", err)
		}
		data, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal user data: %w", err)
		}
		return string(data), nil

	case "whoop://health/recent":
		// Get recent data (last 7 days)
		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -7)

		user, err := s.whoopClient.GetUser()
		if err != nil {
			return "", fmt.Errorf("failed to get user: %w", err)
		}

		userID := user.UserID
		recovery, _ := s.whoopClient.GetRecoveryData(startDate, endDate, &userID)
		sleep, _ := s.whoopClient.GetSleepData(startDate, endDate, &userID)
		workouts, _ := s.whoopClient.GetWorkoutData(startDate, endDate, &userID)

		recentData := map[string]interface{}{
			"recovery": recovery,
			"sleep":    sleep,
			"workouts": workouts,
		}

		data, err := json.MarshalIndent(recentData, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal recent data: %w", err)
		}
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown resource URI: %s", uri)
	}
}

// Helper methods for formatting insights
func (s *MCPServer) getStressRecommendations(stress StressIndicators) string {
	switch stress.StressLevel {
	case "critical":
		return "Immediate intervention recommended. Consider reducing stressors, improving sleep hygiene, and potentially seeking medical evaluation for chronic stress impacts."
	case "high":
		return "Elevated stress levels detected. Focus on stress management techniques, relaxation practices, and identifying primary stressors in therapy."
	case "moderate":
		return "Moderate stress indicators present. Discuss stress management strategies and monitor for progression."
	default:
		return "Stress levels appear within normal range. Continue current coping strategies."
	}
}

func (s *MCPServer) getSleepMentalHealthImplications(analysis SleepAnalysis) string {
	implications := []string{}

	if analysis.AverageHours < 7 {
		implications = append(implications, "Insufficient sleep duration may contribute to mood instability, increased anxiety, and difficulty with emotional regulation")
	}

	if analysis.AverageEfficiency < 0.8 {
		implications = append(implications, "Poor sleep efficiency suggests difficulty maintaining sleep, which can indicate anxiety, stress, or sleep disorders")
	}

	if analysis.SleepQualityTrend == "declining" {
		implications = append(implications, "Declining sleep quality trend may reflect increasing stress, life changes, or developing mental health concerns")
	}

	if len(implications) == 0 {
		return "Sleep patterns appear supportive of mental health and emotional regulation."
	}

	return strings.Join(implications, ". ")
}

func (s *MCPServer) getSleepRecommendations(analysis SleepAnalysis) string {
	recommendations := []string{}

	if analysis.AverageHours < 7 {
		recommendations = append(recommendations, "Focus on extending sleep duration through earlier bedtime and consistent sleep schedule")
	}

	if analysis.AverageEfficiency < 0.85 {
		recommendations = append(recommendations, "Explore sleep hygiene practices and factors affecting sleep maintenance")
	}

	if analysis.ConsistencyScore < 0.7 {
		recommendations = append(recommendations, "Work on sleep schedule consistency to improve circadian rhythm regulation")
	}

	if len(recommendations) == 0 {
		return "Continue current sleep practices as they appear to be supporting good sleep quality."
	}

	return strings.Join(recommendations, "; ")
}

func (s *MCPServer) getActivityBehavioralInsights(patterns ActivityPatterns) string {
	insights := []string{}

	if patterns.WeeklyWorkouts == 0 {
		insights = append(insights, "Lack of recorded physical activity may indicate low motivation, energy, or potential depression symptoms")
	} else if patterns.WeeklyWorkouts > 7 {
		insights = append(insights, "High exercise frequency might indicate compulsive exercise behaviors or use of exercise as primary coping mechanism")
	}

	if patterns.OvertrainingRisk == "high" {
		insights = append(insights, "High training load may contribute to physical and mental fatigue, potentially exacerbating stress and mood issues")
	}

	if patterns.IntensityBalance == "high_intensity_focused" {
		insights = append(insights, "Preference for high-intensity exercise may reflect need for intense stimulation or avoidance behaviors")
	}

	if len(insights) == 0 {
		return "Activity patterns suggest a balanced approach to exercise that likely supports mental health."
	}

	return strings.Join(insights, ". ")
}

func (s *MCPServer) formatRecoveryTrend(trend RecoveryTrend, days int) string {
	return fmt.Sprintf(`# Recovery Trend Analysis (%d days)

## Trend Summary
- **Overall Trend:** %s
- **Average Score:** %.1f%%
- **Weekly Change:** %.1f points
- **Consistency:** %.1f%%

## Recent Scores
%s

## Interpretation
%s`,
		days,
		trend.Trend,
		trend.AverageScore,
		trend.WeeklyChange,
		trend.ConsistencyScore*100,
		s.formatScoreList(trend.LastSevenDays),
		s.interpretRecoveryTrend(trend))
}

func (s *MCPServer) formatSleepTrend(analysis SleepAnalysis, days int) string {
	return fmt.Sprintf(`# Sleep Trend Analysis (%d days)

## Sleep Summary
- **Average Duration:** %.1f hours
- **Sleep Efficiency:** %.1f%%
- **Quality Trend:** %s
- **Consistency:** %.1f%%

## Analysis
%s`,
		days,
		analysis.AverageHours,
		analysis.AverageEfficiency*100,
		analysis.SleepQualityTrend,
		analysis.ConsistencyScore*100,
		s.interpretSleepTrend(analysis))
}

func (s *MCPServer) formatStrainTrend(cycles []WhoopCycle, days int) string {
	if len(cycles) == 0 {
		return "No strain data available for the requested period."
	}

	var strains []float64
	for _, cycle := range cycles {
		strains = append(strains, cycle.Score.Strain)
	}

	avgStrain := 0.0
	if len(strains) > 0 {
		sum := 0.0
		for _, strain := range strains {
			sum += strain
		}
		avgStrain = sum / float64(len(strains))
	}

	return fmt.Sprintf(`# Strain Trend Analysis (%d days)

## Strain Summary
- **Average Strain:** %.1f
- **Total Sessions:** %d
- **Strain Range:** %.1f - %.1f

## Recent Pattern
%s`,
		days,
		avgStrain,
		len(cycles),
		s.findMin(strains),
		s.findMax(strains),
		s.interpretStrainPattern(strains))
}

func (s *MCPServer) formatScoreList(scores []float64) string {
	if len(scores) == 0 {
		return "No recent scores available"
	}

	var formatted []string
	for i, score := range scores {
		formatted = append(formatted, fmt.Sprintf("Day %d: %.1f%%", i+1, score))
	}
	return strings.Join(formatted, ", ")
}

func (s *MCPServer) interpretRecoveryTrend(trend RecoveryTrend) string {
	interpretation := fmt.Sprintf("Recovery is showing a %s trend", trend.Trend)

	if trend.Trend == "declining" {
		interpretation += " which may indicate increasing stress, inadequate recovery, or developing health concerns"
	} else if trend.Trend == "improving" {
		interpretation += " suggesting effective stress management and recovery strategies"
	}

	if trend.ConsistencyScore < 0.6 {
		interpretation += ". High variability in scores suggests inconsistent stressors or recovery practices"
	}

	return interpretation + "."
}

func (s *MCPServer) interpretSleepTrend(analysis SleepAnalysis) string {
	interpretation := ""

	if analysis.AverageHours < 7 {
		interpretation += "Sleep duration is below optimal range for most adults. "
	}

	if analysis.AverageEfficiency < 0.85 {
		interpretation += "Sleep efficiency suggests difficulty maintaining sleep. "
	}

	if analysis.SleepQualityTrend == "declining" {
		interpretation += "Declining quality trend requires attention to identify contributing factors."
	} else if analysis.SleepQualityTrend == "improving" {
		interpretation += "Improving quality trend suggests positive changes in sleep habits or stress management."
	}

	if interpretation == "" {
		interpretation = "Sleep patterns appear to be within healthy ranges."
	}

	return interpretation
}

func (s *MCPServer) interpretStrainPattern(strains []float64) string {
	if len(strains) == 0 {
		return "No strain data to analyze"
	}

	avg := 0.0
	for _, strain := range strains {
		avg += strain
	}
	avg /= float64(len(strains))

	if avg > 15 {
		return "High average strain may indicate intense training that could impact recovery"
	} else if avg < 8 {
		return "Low average strain suggests minimal physical stress, which may be appropriate for recovery phases"
	}

	return "Strain levels appear balanced for maintaining fitness while allowing recovery"
}

func (s *MCPServer) findMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (s *MCPServer) findMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// executeWhoopAuthSetupTool helps users set up Whoop OAuth authentication
func (s *MCPServer) executeWhoopAuthSetupTool(arguments json.RawMessage) (string, error) {
	var input struct {
		ClientID          string `json:"client_id,omitempty"`
		AuthorizationCode string `json:"authorization_code,omitempty"`
		ClientSecret      string `json:"client_secret,omitempty"`
	}

	if err := json.Unmarshal(arguments, &input); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// If only client_id provided, generate authorization URL
	if input.ClientID != "" && input.AuthorizationCode == "" {
		return s.generateAuthURL(input.ClientID), nil
	}

	// If authorization code provided, exchange for tokens
	if input.AuthorizationCode != "" && input.ClientSecret != "" {
		return s.exchangeCodeForTokens(input.ClientID, input.ClientSecret, input.AuthorizationCode)
	}

	// Otherwise, provide general setup instructions
	return s.generateAuthInstructions(), nil
}

// generateAuthURL creates the Whoop OAuth authorization URL
func (s *MCPServer) generateAuthURL(clientID string) string {
	baseURL := "https://api.prod.whoop.com/oauth/oauth2/auth"

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", "http://localhost:3000/callback")
	params.Set("response_type", "code")
	params.Set("scope", "read:recovery read:sleep read:workout read:cycles read:profile offline")
	params.Set("state", "whoop-mcp-auth")

	authURL := baseURL + "?" + params.Encode()

	return fmt.Sprintf(`# Whoop OAuth Setup - Step 1

## 🔗 Authorization URL Generated

**Click this URL to authorize your Whoop app:**

%s

## 📋 Next Steps:

1. **Open the URL above** in your browser
2. **Log in to Whoop** and authorize the app
3. **Copy the authorization code** from the callback URL
4. **Ask me to exchange the code for tokens** by saying:
   "Exchange my Whoop authorization code: [YOUR_CODE_HERE]"

## ⚠️ Note:
The redirect URL may show an error page - that's normal! Just copy the 'code' parameter from the URL bar.

Example callback URL:
http://localhost:3000/callback?code=ABC123...&state=whoop-mcp-auth

Copy everything after "code=" and before "&state".`, authURL)
}

// exchangeCodeForTokens exchanges authorization code for access/refresh tokens
func (s *MCPServer) exchangeCodeForTokens(clientID, clientSecret, authCode string) (string, error) {
	tokenURL := "https://api.prod.whoop.com/oauth/oauth2/token"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", "http://localhost:3000/callback")
	data.Set("code", authCode)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to exchange authorization code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Sprintf(`# ❌ Token Exchange Failed

**Error (status %d):**
%s

## Common Issues:
- Authorization code already used (codes are single-use)
- Authorization code expired (they expire quickly)
- Wrong redirect URI (must match exactly)
- Invalid client credentials

## 🔄 Try Again:
Ask me to generate a new authorization URL with your client_id.`, resp.StatusCode, string(body)), nil
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return fmt.Sprintf(`# ✅ Success! Whoop Tokens Obtained

## 🎉 Your Authentication is Complete!

**Access Token:** %s
**Refresh Token:** %s
**Expires in:** %d seconds (%.1f hours)
**Scopes:** %s

## 🚀 Next Steps:

1. **Update your .env file** with the access token:
   %s
   
2. **Restart Claude Desktop** to load the new token

3. **Test your connection** by asking me:
   "Analyze my Whoop data from yesterday"

## 💡 Pro Tip:
Save the refresh token! It can be used to get new access tokens when the current one expires.`,
		tokenResp.AccessToken,
		tokenResp.RefreshToken,
		tokenResp.ExpiresIn,
		float64(tokenResp.ExpiresIn)/3600,
		tokenResp.Scope,
		fmt.Sprintf("WHOOP_API_KEY=%s", tokenResp.AccessToken)), nil
}

// generateAuthInstructions provides general setup instructions
func (s *MCPServer) generateAuthInstructions() string {
	return `# 🔐 Whoop OAuth Setup Guide

## Prerequisites:
1. **Whoop Developer Account** - Sign up at https://developer.whoop.com
2. **Create an App** in the Whoop Developer Portal
3. **Set Redirect URI** to: http://localhost:3000/callback

## Setup Process:

### Step 1: Get Authorization URL
Ask me: "Generate Whoop auth URL for client_id: YOUR_CLIENT_ID"

### Step 2: Authorize App
I'll provide a URL to authorize your app with Whoop

### Step 3: Exchange Code
After authorization, ask me: "Exchange Whoop code: YOUR_CODE with secret: YOUR_SECRET"

### Step 4: Update Configuration
I'll provide the access token to add to your .env file

## 💡 Need Help?
- Ask me to "Generate Whoop auth URL" if you have a client_id
- Visit https://developer.whoop.com for app setup instructions
- The process takes just a few minutes!`
}
