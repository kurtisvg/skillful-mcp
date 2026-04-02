package cmd

import (
	"flag"
	"os"
	"testing"
)

func TestDefaultConfigFlag(t *testing.T) {
	// Reset flags for test isolation
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configPath = ""
	flag.StringVar(&configPath, "config", "./mcp.json", "Path to MCP config file")
	if err := flag.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}

	if configPath != "./mcp.json" {
		t.Errorf("expected default config path './mcp.json', got %q", configPath)
	}
}

func TestCustomConfigFlag(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configPath = ""
	flag.StringVar(&configPath, "config", "./mcp.json", "Path to MCP config file")
	if err := flag.CommandLine.Parse([]string{"--config", "/tmp/custom.json"}); err != nil {
		t.Fatal(err)
	}

	if configPath != "/tmp/custom.json" {
		t.Errorf("expected config path '/tmp/custom.json', got %q", configPath)
	}
}
