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

const usage = `ob1-go — personal memory MCP server

Usage: ob1-go [flags]

Flags:
  --db PATH               SQLite file path (env DB_PATH)
  --mcp-access-key KEY    auth key clients send as x-brain-key (env MCP_ACCESS_KEY)
  --env PATH              read KEY=VALUE lines from PATH before applying config
  -h, --help              show this help

MCP_ACCESS_KEY is required, by flag, environment, or --env file. Everything
else has a default. Precedence: flag, then environment, then --env file.

Environment (default):
  MCP_ACCESS_KEY                 required, no default
  DB_PATH (/data/openbrain.db)   SQLite file path
  PORT (8081)                    HTTP listen port
  EMBEDDING_API_BASE (http://127.0.0.1:11434/v1)
  EMBEDDING_API_KEY (ollama)
  EMBEDDING_MODEL (nomic-embed-text)
  CHAT_API_BASE (EMBEDDING_API_BASE)
  CHAT_API_KEY (EMBEDDING_API_KEY)
  CHAT_MODEL (qwen2.5:1.5b)      model used for metadata extraction
  OPEN_BRAIN_CITATION_BASE_URL (https://openbrain.local/thoughts)
`

type flags struct {
	dbPath       string
	mcpAccessKey string
	envFile      string
	help         bool
}

func parseFlags(args []string) (*flags, error) {
	f := &flags{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" || arg == "help" {
			f.help = true
			return f, nil
		}

		name, value, inline := strings.Cut(arg, "=")
		long, ok := strings.CutPrefix(name, "--")
		if !ok {
			return nil, fmt.Errorf("unknown flag %q (long flags take two dashes, e.g. --db)", arg)
		}

		var target *string
		switch long {
		case "db":
			target = &f.dbPath
		case "mcp-access-key":
			target = &f.mcpAccessKey
		case "env":
			target = &f.envFile
		default:
			return nil, fmt.Errorf("unknown flag %q", arg)
		}

		if !inline {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag %s needs a value", arg)
			}
			i++
			value = args[i]
		}
		*target = value
	}
	return f, nil
}

// loadEnvFile applies KEY=VALUE lines without overwriting variables that are
// already set, so an explicit environment always beats the file.
func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig(f *flags) (*Config, error) {
	if f.envFile != "" {
		if err := loadEnvFile(f.envFile); err != nil {
			return nil, fmt.Errorf("reading %s: %w", f.envFile, err)
		}
	}

	accessKey := f.mcpAccessKey
	if accessKey == "" {
		accessKey = os.Getenv("MCP_ACCESS_KEY")
	}
	if accessKey == "" {
		return nil, fmt.Errorf("MCP_ACCESS_KEY is required")
	}

	embeddingAPIBase := getEnv("EMBEDDING_API_BASE", "http://127.0.0.1:11434/v1")
	embeddingAPIKey := getEnv("EMBEDDING_API_KEY", "ollama")

	port := 8081
	if p := os.Getenv("PORT"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT %q: %w", p, err)
		}
		port = v
	}

	citationBase := getEnv("OPEN_BRAIN_CITATION_BASE_URL", "https://openbrain.local/thoughts")
	citationBase = strings.TrimSuffix(citationBase, "/")

	dbPath := f.dbPath
	if dbPath == "" {
		dbPath = getEnv("DB_PATH", "/data/openbrain.db")
	}

	return &Config{
		DBPath:           dbPath,
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
	f, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprint(os.Stderr, "\n", usage)
		os.Exit(1)
	}
	if f.help {
		fmt.Print(usage)
		return
	}

	cfg, err := loadConfig(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		fmt.Fprint(os.Stderr, "\n", usage)
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
		registerDeleteThoughtTool(server, store)
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
