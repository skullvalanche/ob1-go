package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const metadataSystemPrompt = `Extract metadata from the user's captured thought. Return JSON with:
- "people": array of people mentioned (empty if none)
- "action_items": array of implied to-dos (empty if none)
- "dates_mentioned": array of dates YYYY-MM-DD (empty if none)
- "topics": array of 1-3 short topic tags (always at least one)
- "type": one of "observation", "task", "idea", "reference", "person_note"
Only extract what's explicitly there.`

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func getEmbedding(ctx context.Context, cfg *Config, text string) ([]float32, error) {
	reqBody, _ := json.Marshal(map[string]any{"model": cfg.EmbeddingModel, "input": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.EmbeddingAPIBase+"/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.EmbeddingAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API failed: %d %s", resp.StatusCode, respBody)
	}

	var parsed embeddingResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("embedding API returned no data")
	}
	return parsed.Data[0].Embedding, nil
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func extractMetadata(ctx context.Context, cfg *Config, text string) (map[string]any, error) {
	fallback := map[string]any{"topics": []string{"uncategorized"}, "type": "observation"}

	reqBody, _ := json.Marshal(map[string]any{
		"model":           cfg.ChatModel,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": metadataSystemPrompt},
			{"role": "user", "content": text},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.ChatAPIBase+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.ChatAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat API failed: %d %s", resp.StatusCode, respBody)
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return fallback, nil
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &meta); err != nil {
		return fallback, nil
	}
	return meta, nil
}
