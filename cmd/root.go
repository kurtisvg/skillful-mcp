package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"skillful-mcp/internal/config"
	"skillful-mcp/internal/mcpserver"
	"skillful-mcp/internal/server"
)

var (
	configPath string
	transport  string
	host       string
	port       string
)

func init() {
	flag.StringVar(&configPath, "config", "./mcp.json", "Path to MCP config file")
	flag.StringVar(&transport, "transport", "stdio", "Upstream transport: stdio or http")
	flag.StringVar(&host, "host", "localhost", "HTTP host (when transport=http)")
	flag.StringVar(&port, "port", "8080", "HTTP port (when transport=http)")
}

func Execute() {
	flag.Parse()

	servers, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Loaded %d server(s):\n", len(servers))
	for name, srv := range servers {
		switch s := srv.(type) {
		case *config.StdioServer:
			fmt.Fprintf(os.Stderr, "  [%s] stdio → %s %v\n", name, s.Command, s.Args)
		case *config.HTTPServer:
			fmt.Fprintf(os.Stderr, "  [%s] http → %s\n", name, s.URL)
		case *config.SSEServer:
			fmt.Fprintf(os.Stderr, "  [%s] sse → %s\n", name, s.URL)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	mgr, err := mcpserver.NewManager(ctx, servers)
	if err != nil {
		log.Fatalf("Failed to connect to servers: %v", err)
	}
	defer mgr.Close()

	fmt.Fprintf(os.Stderr, "Connected to %d skill(s): %v\n", len(mgr.ListServerNames()), mgr.ListServerNames())

	s := server.NewServer(mgr)
	if err := server.Serve(ctx, s, transport, host, port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
