package app

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"skillful-mcp/internal/mcpserver"
	"skillful-mcp/internal/tools"
	"skillful-mcp/internal/version"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates an MCP server with all tools registered.
func NewServer(mgr *mcpserver.Manager) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "skillful-mcp",
		Version: version.Version,
	}, nil)

	tools.RegisterListSkills(s, mgr)
	tools.RegisterUseSkill(s, mgr)
	tools.RegisterReadResource(s, mgr)
	tools.RegisterExecuteCode(s, mgr)

	return s
}

// ServeStdio runs the MCP server over stdin/stdout.
func ServeStdio(ctx context.Context, s *mcp.Server) error {
	return s.Run(ctx, &mcp.StdioTransport{})
}

// ServeHTTP runs the MCP server over HTTP with graceful shutdown.
func ServeHTTP(ctx context.Context, s *mcp.Server, host, port string) error {
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s
	}, nil)
	addr := net.JoinHostPort(host, port)
	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Warn("http server shutdown error", "error", err)
		}
	}()
	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
