package cmd

import (
	"context"
	"flag"
	"log"
	"log/slog"
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

	slog.Info("loaded config", "servers", len(servers))
	for name, srv := range servers {
		switch s := srv.(type) {
		case *config.StdioServer:
			slog.Info("configured server", "name", name, "transport", "stdio", "command", s.Command, "args", s.Args)
		case *config.HTTPServer:
			slog.Info("configured server", "name", name, "transport", "http", "url", s.URL)
		case *config.SSEServer:
			slog.Info("configured server", "name", name, "transport", "sse", "url", s.URL)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	mgr, err := mcpserver.NewManager(ctx, servers)
	if err != nil {
		log.Fatalf("Failed to connect to servers: %v", err)
	}
	defer mgr.Close()

	slog.Info("connected to skills", "skills", mgr.ListServerNames())

	s := server.NewServer(mgr)
	if err := server.Serve(ctx, s, transport, host, port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
