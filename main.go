package main

import (
	"log"
	"os"
)

func main() {
	// Set up logging to stderr to avoid interfering with stdio communication
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Create and start the MCP server
	server, err := NewMCPServer()
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	log.Println("Starting Whoop MCP Server...")
	log.Println("Server ready to accept JSON-RPC 2.0 requests via stdio")

	// Run the server (blocks until stdin is closed)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Whoop MCP Server shutting down")
}
