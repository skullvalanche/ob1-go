package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	DBPath           string
	EmbeddingAPIBase string
	EmbeddingAPIKey  string
	EmbeddingModel   string
	ChatAPIBase      string
	ChatAPIKey       string
	ChatModel        string
	MCPAccessKey     string
	CitationBaseURL  string
	Port             int
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() (*Config, error) {
	accessKey := os.Getenv("MCP_ACCESS_KEY")
	if accessKey == "" {
		return nil, fmt.Errorf("MCP_ACCESS_KEY is required")
	}

	embeddingAPIBase := getEnv("EMBEDDING_API_BASE", "http://127.0.0.1:11434/v1")
	embeddingAPIKey := getEnv("EMBEDDING_API_KEY", "ollama")

	port := 8000
	if p := os.Getenv("PORT"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT %q: %w", p, err)
		}
		port = v
	}

	citationBase := getEnv("OPEN_BRAIN_CITATION_BASE_URL", "https://openbrain.local/thoughts")
	citationBase = strings.TrimSuffix(citationBase, "/")

	return &Config{
		DBPath:           getEnv("DB_PATH", "/data/openbrain.db"),
		EmbeddingAPIBase: embeddingAPIBase,
		EmbeddingAPIKey:  embeddingAPIKey,
		EmbeddingModel:   getEnv("EMBEDDING_MODEL", "nomic-embed-text"),
		ChatAPIBase:      getEnv("CHAT_API_BASE", embeddingAPIBase),
		ChatAPIKey:       getEnv("CHAT_API_KEY", embeddingAPIKey),
		ChatModel:        getEnv("CHAT_MODEL", "qwen2.5:1.5b"),
		MCPAccessKey:     accessKey,
		CitationBaseURL:  citationBase,
		Port:             port,
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	store, err := openStore(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db error:", err)
		os.Exit(1)
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		server := mcp.NewServer(&mcp.Implementation{Name: "ob1-go", Version: "1.0.0"}, nil)
		registerReadOnlyTools(server, store, cfg)
		registerCaptureThoughtTool(server, store, cfg)
		return server
	}, nil)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-brain-key")
		if key == "" {
			key = r.URL.Query().Get("key")
		}
		if key != cfg.MCPAccessKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		mcpHandler.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("ob1-go listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
