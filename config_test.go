package main

import (
	"os"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	os.Setenv("MCP_ACCESS_KEY", "test-key")
	defer os.Unsetenv("MCP_ACCESS_KEY")
	for _, k := range []string{"DB_PATH", "EMBEDDING_API_BASE", "EMBEDDING_API_KEY", "EMBEDDING_MODEL", "CHAT_API_BASE", "CHAT_API_KEY", "CHAT_MODEL", "OPEN_BRAIN_CITATION_BASE_URL", "PORT"} {
		os.Unsetenv(k)
	}

	cfg, err := loadConfig()
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
	if _, err := loadConfig(); err == nil {
		t.Fatal("loadConfig() expected error when MCP_ACCESS_KEY unset, got nil")
	}
}

func TestLoadConfigStripsCitationTrailingSlash(t *testing.T) {
	os.Setenv("MCP_ACCESS_KEY", "test-key")
	os.Setenv("OPEN_BRAIN_CITATION_BASE_URL", "https://example.com/thoughts/")
	defer os.Unsetenv("MCP_ACCESS_KEY")
	defer os.Unsetenv("OPEN_BRAIN_CITATION_BASE_URL")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.CitationBaseURL != "https://example.com/thoughts" {
		t.Errorf("CitationBaseURL = %q, want trailing slash stripped", cfg.CitationBaseURL)
	}
}
