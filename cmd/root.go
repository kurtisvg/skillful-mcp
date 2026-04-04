package cmd

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"skillful-mcp/internal/app"
	"skillful-mcp/internal/config"
	"skillful-mcp/internal/mcpserver"
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
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
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
		slog.Error("failed to connect to servers", "error", err)
		os.Exit(1)
	}
	defer mgr.Close()

	slog.Info("connected to skills", "skills", mgr.ListServerNames())

	s := app.NewServer(mgr)
	var serveErr error
	switch transport {
	case "stdio":
		serveErr = app.ServeStdio(ctx, s)
	case "http":
		serveErr = app.ServeHTTP(ctx, s, host, port)
	default:
		slog.Error("unknown transport (use 'stdio' or 'http')", "transport", transport)
		os.Exit(1)
	}
	if serveErr != nil {
		slog.Error("server error", "error", serveErr)
		os.Exit(1)
	}
}
