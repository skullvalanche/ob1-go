package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetEmbeddingSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "nomic-embed-text" {
			t.Errorf("model = %v", body["model"])
		}
		w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer srv.Close()

	cfg := &Config{EmbeddingAPIBase: srv.URL, EmbeddingAPIKey: "test-key", EmbeddingModel: "nomic-embed-text"}
	emb, err := getEmbedding(context.Background(), cfg, "hello")
	if err != nil {
		t.Fatalf("getEmbedding: %v", err)
	}
	if len(emb) != 3 || emb[0] != 0.1 {
		t.Errorf("emb = %v", emb)
	}
}

func TestGetEmbeddingHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	cfg := &Config{EmbeddingAPIBase: srv.URL, EmbeddingAPIKey: "k", EmbeddingModel: "m"}
	_, err := getEmbedding(context.Background(), cfg, "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want it to mention status 500", err)
	}
}

func TestExtractMetadataSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"type\":\"task\",\"topics\":[\"work\"]}"}}]}`))
	}))
	defer srv.Close()

	cfg := &Config{ChatAPIBase: srv.URL, ChatAPIKey: "k", ChatModel: "m"}
	meta, err := extractMetadata(context.Background(), cfg, "remember to do X")
	if err != nil {
		t.Fatalf("extractMetadata: %v", err)
	}
	if meta["type"] != "task" {
		t.Errorf("meta[type] = %v", meta["type"])
	}
}

func TestExtractMetadataFallbackOnBadModelJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
	}))
	defer srv.Close()

	cfg := &Config{ChatAPIBase: srv.URL, ChatAPIKey: "k", ChatModel: "m"}
	meta, err := extractMetadata(context.Background(), cfg, "text")
	if err != nil {
		t.Fatalf("extractMetadata: %v", err)
	}
	if meta["type"] != "observation" {
		t.Errorf("meta[type] = %v, want fallback observation", meta["type"])
	}
}

func TestExtractMetadataHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	cfg := &Config{ChatAPIBase: srv.URL, ChatAPIKey: "k", ChatModel: "m"}
	_, err := extractMetadata(context.Background(), cfg, "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want it to mention status 500", err)
	}
}

func TestExtractMetadataPropagatesTransportError(t *testing.T) {
	cfg := &Config{ChatAPIBase: "http://127.0.0.1:1", ChatAPIKey: "k", ChatModel: "m"}
	_, err := extractMetadata(context.Background(), cfg, "text")
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
}
