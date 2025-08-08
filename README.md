# Whoop Health Data MCP Server

A comprehensive Model Context Protocol (MCP) server that integrates with the **Whoop API V2** to provide health data analysis for AI therapy applications.

> **✅ V2 Ready**: This server uses the latest Whoop API V2 with UUID identifiers and enhanced data structures. V1 will be deprecated October 1, 2025.

## Features

### 🏥 Whoop API V2 Integration
- **UUID-based IDs**: Future-proof identifiers for all health data
- **Enhanced Sleep Data**: Performance percentages, consistency metrics, respiratory rate
- **Improved Workouts**: Sport names, detailed zone durations, altitude changes
- **Real-time Data**: Recovery, sleep, strain, and workout data from Whoop

### 🧠 AI Therapy Enhancement
- **Therapy-Focused Analysis**: Mental health indicators and stress patterns
- **Red Flag Detection**: Automatic identification of concerning health patterns
- **Stress Indicators**: Physiological stress markers from HRV and heart rate
- **Sleep Impact Analysis**: Sleep quality patterns and mental health implications

### 🔧 Technical Excellence
- **MCP Protocol**: Standard integration with Claude and other AI assistants
- **Privacy-First**: Secure data handling with no persistent storage
- **Rate Limited**: Respects Whoop API limits with intelligent caching
- **Concurrent Processing**: Parallel API calls for optimal performance

## Quick Start

1.**Setup Environment**:

    ```bash
    cp .env.example .env
    # Add your WHOOP_API_KEY to .env
    ```

2.**Build and Run**:

    ```bash
    go mod tidy
    go build -o bin/whoop-mcp-server
    ./bin/whoop-mcp-server
    ```

3.**Configure Claude Desktop**:

    ```json
    {
      "mcpServers": {
        "whoop-health": {
          "command": "/path/to/bin/whoop-mcp-server"
        }
      }
    }
    ```

## Available Tools

get_health_summary: Comprehensive health overview for therapy
get_recovery_data: Detailed recovery metrics and trends
get_sleep_analysis: Sleep quality analysis for mental health
get_stress_indicators: Physiological stress markers
get_activity_patterns: Exercise and activity behavioral insights

## API Integration

This server integrates with Whoop API v1. You'll need:

- Whoop account with developer access
- Valid API key from Whoop Developer Portal
- Active Whoop device with recent data

## Development

    ```bash
    # Run tests
    make test

    # Run with auto-reload
    make dev

    # Build for production
    make build-prod
    ```

## Privacy & Security

- No persistent data storage
- API keys stored in environment variables
- Health data never logged or cached permanently
- Designed with HIPAA-style privacy considerations

## Development Prompts

    ```markdown
    # When working on this project, remember:

    ## For API Integration:
    "I'm working on Whoop API integration for an MCP server. Help me implement [specific endpoint] with proper error handling, rate limiting, and therapy-relevant data extraction."

    ## For Health Analysis:
    "I need to analyze Whoop health data for therapeutic insights. Help me implement analysis that identifies [stress patterns/sleep issues/recovery trends] and generates actionable recommendations for therapy sessions."

    ## For MCP Protocol:
    "I'm implementing MCP protocol tools for health data. Help me create a tool that [specific functionality] with proper JSON schema validation and error handling following MCP standards."

    ## For Testing:
    "Help me write comprehensive tests for my Whoop MCP server, including unit tests for [specific component] and integration tests with mocked Whoop API responses."

    ## For Data Privacy:
    "I need to ensure my health data MCP server maintains privacy and security. Help me implement [specific feature] while following best practices for sensitive health information."
    ```
