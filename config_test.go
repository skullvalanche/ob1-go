package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFlags(t *testing.T) {
	f, err := parseFlags([]string{"--db", "/tmp/a.db", "--env=/tmp/e", "--mcp-access-key", "k"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if f.dbPath != "/tmp/a.db" || f.envFile != "/tmp/e" || f.mcpAccessKey != "k" {
		t.Errorf("flags = %+v", f)
	}

	if f, err := parseFlags([]string{"--help"}); err != nil || !f.help {
		t.Errorf("parseFlags(--help) = %+v, %v", f, err)
	}
	if _, err := parseFlags([]string{"--nope"}); err == nil {
		t.Error("parseFlags(--nope) expected error")
	}
	if _, err := parseFlags([]string{"--db"}); err == nil {
		t.Error("parseFlags(--db) with no value expected error")
	}
	if _, err := parseFlags([]string{"-db", "/tmp/a.db"}); err == nil {
		t.Error("parseFlags(-db) expected error: long flags need two dashes")
	}
}

func TestLoadEnvFileDoesNotOverrideEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	body := "# comment\n\nexport MCP_ACCESS_KEY=\"from-file\"\nCHAT_MODEL='file-model'\nnonsense\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	os.Setenv("MCP_ACCESS_KEY", "from-env")
	os.Unsetenv("CHAT_MODEL")
	defer os.Unsetenv("MCP_ACCESS_KEY")
	defer os.Unsetenv("CHAT_MODEL")

	if err := loadEnvFile(path); err != nil {
		t.Fatalf("loadEnvFile: %v", err)
	}
	if got := os.Getenv("MCP_ACCESS_KEY"); got != "from-env" {
		t.Errorf("MCP_ACCESS_KEY = %q, want existing env to win", got)
	}
	if got := os.Getenv("CHAT_MODEL"); got != "file-model" {
		t.Errorf("CHAT_MODEL = %q, want value from file", got)
	}

	if err := loadEnvFile(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("loadEnvFile(missing) expected error")
	}
}

func TestLoadConfigFlagsBeatEnv(t *testing.T) {
	os.Setenv("MCP_ACCESS_KEY", "env-key")
	os.Setenv("DB_PATH", "/env/path.db")
	defer os.Unsetenv("MCP_ACCESS_KEY")
	defer os.Unsetenv("DB_PATH")

	cfg, err := loadConfig(&flags{dbPath: "/flag/path.db", mcpAccessKey: "flag-key"})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.DBPath != "/flag/path.db" {
		t.Errorf("DBPath = %q, want flag to win", cfg.DBPath)
	}
	if cfg.MCPAccessKey != "flag-key" {
		t.Errorf("MCPAccessKey = %q, want flag to win", cfg.MCPAccessKey)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	os.Setenv("MCP_ACCESS_KEY", "test-key")
	defer os.Unsetenv("MCP_ACCESS_KEY")
	for _, k := range []string{"DB_PATH", "EMBEDDING_API_BASE", "EMBEDDING_API_KEY", "EMBEDDING_MODEL", "CHAT_API_BASE", "CHAT_API_KEY", "CHAT_MODEL", "OPEN_BRAIN_CITATION_BASE_URL", "PORT"} {
		os.Unsetenv(k)
	}

	cfg, err := loadConfig(&flags{})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.DBPath != "/data/openbrain.db" {
		t.Errorf("DBPath = %q, want /data/openbrain.db", cfg.DBPath)
	}
	if cfg.EmbeddingAPIBase != "http://127.0.0.1:11434/v1" {
		t.Errorf("EmbeddingAPIBase = %q", cfg.EmbeddingAPIBase)
	}
	if cfg.EmbeddingAPIKey != "ollama" {
		t.Errorf("EmbeddingAPIKey = %q", cfg.EmbeddingAPIKey)
	}
	if cfg.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("EmbeddingModel = %q", cfg.EmbeddingModel)
	}
	if cfg.ChatAPIBase != cfg.EmbeddingAPIBase {
		t.Errorf("ChatAPIBase = %q, want fallback to EmbeddingAPIBase", cfg.ChatAPIBase)
	}
	if cfg.ChatAPIKey != cfg.EmbeddingAPIKey {
		t.Errorf("ChatAPIKey = %q, want fallback to EmbeddingAPIKey", cfg.ChatAPIKey)
	}
	if cfg.ChatModel != "qwen2.5:1.5b" {
		t.Errorf("ChatModel = %q", cfg.ChatModel)
	}
	if cfg.CitationBaseURL != "https://openbrain.local/thoughts" {
		t.Errorf("CitationBaseURL = %q", cfg.CitationBaseURL)
	}
	if cfg.Port != 8081 {
		t.Errorf("Port = %d, want 8081", cfg.Port)
	}
}

func TestLoadConfigRequiresAccessKey(t *testing.T) {
	os.Unsetenv("MCP_ACCESS_KEY")
	if _, err := loadConfig(&flags{}); err == nil {
		t.Fatal("loadConfig() expected error when MCP_ACCESS_KEY unset, got nil")
	}
}

func TestLoadConfigStripsCitationTrailingSlash(t *testing.T) {
	os.Setenv("MCP_ACCESS_KEY", "test-key")
	os.Setenv("OPEN_BRAIN_CITATION_BASE_URL", "https://example.com/thoughts/")
	defer os.Unsetenv("MCP_ACCESS_KEY")
	defer os.Unsetenv("OPEN_BRAIN_CITATION_BASE_URL")

	cfg, err := loadConfig(&flags{})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.CitationBaseURL != "https://example.com/thoughts" {
		t.Errorf("CitationBaseURL = %q, want trailing slash stripped", cfg.CitationBaseURL)
	}
}
