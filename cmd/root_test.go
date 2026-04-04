package cmd

import (
	"testing"
)

func TestDefaultFlags(t *testing.T) {
	t.Parallel()
	opts := parseFlags([]string{})

	if opts.configPath != "./mcp.json" {
		t.Errorf("configPath = %q, want './mcp.json'", opts.configPath)
	}
	if opts.transport != "stdio" {
		t.Errorf("transport = %q, want 'stdio'", opts.transport)
	}
	if opts.host != "localhost" {
		t.Errorf("host = %q, want 'localhost'", opts.host)
	}
	if opts.port != "8080" {
		t.Errorf("port = %q, want '8080'", opts.port)
	}
	if opts.version {
		t.Error("version = true, want false")
	}
}

func TestCustomFlags(t *testing.T) {
	t.Parallel()
	opts := parseFlags([]string{
		"--config", "/tmp/custom.json",
		"--transport", "http",
		"--host", "0.0.0.0",
		"--port", "9090",
	})

	if opts.configPath != "/tmp/custom.json" {
		t.Errorf("configPath = %q, want '/tmp/custom.json'", opts.configPath)
	}
	if opts.transport != "http" {
		t.Errorf("transport = %q, want 'http'", opts.transport)
	}
	if opts.host != "0.0.0.0" {
		t.Errorf("host = %q, want '0.0.0.0'", opts.host)
	}
	if opts.port != "9090" {
		t.Errorf("port = %q, want '9090'", opts.port)
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()
	opts := parseFlags([]string{"--version"})

	if !opts.version {
		t.Error("version = false, want true")
	}
}
