package clientmanager

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"sort"

	"skillful-mcp/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Manager struct {
	sessions map[string]*mcp.ClientSession
}

// ConnectAll creates a Manager by connecting to all servers in the config.
func ConnectAll(ctx context.Context, cfg *config.Config) (*Manager, error) {
	m := &Manager{sessions: make(map[string]*mcp.ClientSession)}

	for name, srv := range cfg.MCPServers {
		session, err := connect(ctx, name, &srv)
		if err != nil {
			// Close any sessions we already opened before returning.
			m.Close()
			return nil, fmt.Errorf("connecting to %q: %w", name, err)
		}
		m.sessions[name] = session
		tt, _ := srv.TransportType() // already validated
		slog.Info("connected to server", "skill", name, "transport", tt)
	}

	return m, nil
}

// NewFromSessions creates a Manager from pre-built sessions (useful for testing).
func NewFromSessions(sessions map[string]*mcp.ClientSession) *Manager {
	return &Manager{sessions: sessions}
}

func connect(ctx context.Context, name string, srv *config.ServerConfig) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "skillful-mcp",
		Version: "0.1.0",
	}, nil)

	var transport mcp.Transport

	tt, err := srv.TransportType()
	if err != nil {
		return nil, err
	}

	switch tt {
	case config.TransportSTDIO:
		cmd := exec.Command(srv.Command, srv.Args...)
		cmd.Env = toEnv(srv.Env)
		transport = &mcp.CommandTransport{Command: cmd}

	case config.TransportHTTP:
		httpClient := httpClientWithHeaders(srv.Headers)
		transport = &mcp.StreamableClientTransport{
			Endpoint:   srv.URL,
			HTTPClient: httpClient,
		}

	case config.TransportSSE:
		httpClient := httpClientWithHeaders(srv.Headers)
		transport = &mcp.SSEClientTransport{
			Endpoint:   srv.URL,
			HTTPClient: httpClient,
		}
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (m *Manager) GetSession(name string) (*mcp.ClientSession, error) {
	s, ok := m.sessions[name]
	if !ok {
		return nil, fmt.Errorf("unknown skill: %q", name)
	}
	return s, nil
}

func (m *Manager) ListServerNames() []string {
	names := make([]string, 0, len(m.sessions))
	for name := range m.sessions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *Manager) Close() {
	for name, s := range m.sessions {
		if err := s.Close(); err != nil {
			slog.Warn("error closing session", "skill", name, "error", err)
		}
	}
}

// toEnv converts the configured env map to a slice for exec.Cmd.
// Only the explicitly specified vars are passed to the child process.
// If no env vars are configured, returns nil (child inherits nothing).
func toEnv(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

// headerTransport injects custom HTTP headers into every request.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return t.base.RoundTrip(req)
}

func httpClientWithHeaders(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return http.DefaultClient
	}
	return &http.Client{
		Transport: &headerTransport{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}
}
