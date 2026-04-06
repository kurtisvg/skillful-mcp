package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kurtisvg/skillful-mcp/internal/app"
	"github.com/kurtisvg/skillful-mcp/internal/config"
	"github.com/kurtisvg/skillful-mcp/internal/mcpserver"
	"github.com/kurtisvg/skillful-mcp/internal/version"

	flag "github.com/spf13/pflag"
)

type options struct {
	configPath string
	transport  string
	host       string
	port       string
	version    bool
}

func parseFlags(args []string) options {
	var opts options
	fs := flag.NewFlagSet("skillful-mcp", flag.ExitOnError)
	fs.StringVar(&opts.configPath, "config", "./mcp.json", "Path to MCP config file")
	fs.StringVar(&opts.transport, "transport", "stdio", "Upstream transport: stdio or http")
	fs.StringVar(&opts.host, "host", "localhost", "HTTP host (when transport=http)")
	fs.StringVar(&opts.port, "port", "8080", "HTTP port (when transport=http)")
	fs.BoolVar(&opts.version, "version", false, "Print version and exit")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	return opts
}

func Execute() {
	opts := parseFlags(os.Args[1:])

	if opts.version {
		fmt.Println(version.Version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	servers, err := config.Load(opts.configPath)
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

	mgr, err := mcpserver.NewManager(ctx, servers)
	if err != nil {
		slog.Error("failed to connect to servers", "error", err)
		os.Exit(1)
	}
	defer mgr.Close()

	slog.Info("connected to servers", "servers", mgr.ListServerNames())

	s := app.NewServer(mgr)
	var serveErr error
	switch opts.transport {
	case "stdio":
		serveErr = app.ServeStdio(ctx, s)
	case "http":
		serveErr = app.ServeHTTP(ctx, s, opts.host, opts.port)
	default:
		slog.Error("unknown transport (use 'stdio' or 'http')", "transport", opts.transport)
		os.Exit(1)
	}
	if serveErr != nil {
		slog.Error("server error", "error", serveErr)
		os.Exit(1)
	}
}
