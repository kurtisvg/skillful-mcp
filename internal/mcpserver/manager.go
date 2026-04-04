package mcpserver

import (
	"context"
	"fmt"
	"log/slog"

	"skillful-mcp/internal/config"
)

type Manager struct {
	servers map[string]*Server
	tools   []Tool
}

// NewManager creates a Manager by connecting to all servers in the config.
func NewManager(ctx context.Context, cfgs map[string]config.Server) (*Manager, error) {
	m := &Manager{servers: make(map[string]*Server)}

	for name, srv := range cfgs {
		s, err := NewServer(ctx, srv)
		if err != nil {
			// Close any servers we already opened before returning.
			m.Close()
			return nil, fmt.Errorf("connecting to %q: %w", name, err)
		}
		m.servers[name] = s
		slog.Info("connected to server", "skill", name)
	}

	tools, err := resolveTools(m.servers)
	if err != nil {
		m.Close()
		return nil, err
	}
	m.tools = tools
	return m, nil
}

// NewManagerFromServers creates a Manager from pre-built Servers (useful for testing).
func NewManagerFromServers(servers map[string]*Server) (*Manager, error) {
	m := &Manager{servers: servers}
	tools, err := resolveTools(m.servers)
	if err != nil {
		return nil, err
	}
	m.tools = tools
	return m, nil
}

func (m *Manager) GetServer(name string) (*Server, error) {
	s, ok := m.servers[name]
	if !ok {
		return nil, fmt.Errorf("unknown skill: %q", name)
	}
	return s, nil
}

func (m *Manager) ListServerNames() []string {
	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

func (m *Manager) AllTools() []Tool {
	return m.tools
}

func (m *Manager) ServerTools(name string) []Tool {
	var tools []Tool
	for _, t := range m.tools {
		if t.SkillName == name {
			tools = append(tools, t)
		}
	}
	return tools
}

func (m *Manager) Close() {
	for name, s := range m.servers {
		if err := s.Close(); err != nil {
			slog.Warn("error closing server", "skill", name, "error", err)
		}
	}
}
