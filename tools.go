package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func textResult(text string, isError ...bool) *mcp.CallToolResult {
	res := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
	if len(isError) > 0 && isError[0] {
		res.IsError = true
	}
	return res
}

func errorResult(err error) *mcp.CallToolResult {
	return textResult(fmt.Sprintf("Error: %s", err.Error()), true)
}

func formatDate(createdAt string) string {
	t, err := time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		return createdAt
	}
	return t.Format("2006-01-02")
}

func thoughtTitle(content, createdAt string) string {
	firstLine := strings.Join(strings.Fields(content), " ")
	if len(firstLine) > 80 {
		firstLine = firstLine[:80]
	}
	datePrefix := "Open Brain"
	if createdAt != "" {
		datePrefix = formatDate(createdAt)
	}
	if firstLine != "" {
		return datePrefix + " - " + firstLine
	}
	return datePrefix + " thought"
}

func thoughtURL(cfg *Config, id int64) string {
	return fmt.Sprintf("%s/%d", cfg.CitationBaseURL, id)
}

func stringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

type kv struct {
	key   string
	count int
}

func topN(m map[string]int, n int) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].count > out[j].count })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

type searchResult struct {
	ID         int64
	Content    string
	CreatedAt  string
	Similarity float64
	Meta       map[string]any
}

func runSearch(ctx context.Context, store *Store, cfg *Config, query string, limit int, threshold float64) ([]searchResult, error) {
	qEmb, err := getEmbedding(ctx, cfg, query)
	if err != nil {
		return nil, err
	}
	rows, err := store.AllForSearch()
	if err != nil {
		return nil, err
	}

	results := make([]searchResult, 0, len(rows))
	for _, r := range rows {
		sim := cosineSimilarity(decodeEmbedding(r.Embedding), qEmb)
		if sim < threshold {
			continue
		}
		var meta map[string]any
		json.Unmarshal([]byte(r.Metadata), &meta)
		results = append(results, searchResult{ID: r.ID, Content: r.Content, CreatedAt: r.CreatedAt, Similarity: sim, Meta: meta})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Similarity > results[j].Similarity })
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

type searchParams struct {
	Query string `json:"query" jsonschema:"The search query"`
}

type fetchParams struct {
	ID string `json:"id" jsonschema:"Thought ID returned by search"`
}

type searchThoughtsParams struct {
	Query     string  `json:"query" jsonschema:"What to search for"`
	Limit     int     `json:"limit,omitempty" jsonschema:"Max results (default 10)"`
	Threshold float64 `json:"threshold,omitempty" jsonschema:"Similarity threshold 0-1 (default 0.5)"`
}

type listThoughtsParams struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"Max results (default 10)"`
	Type   string `json:"type,omitempty" jsonschema:"Filter by type: observation, task, idea, reference, person_note"`
	Topic  string `json:"topic,omitempty" jsonschema:"Filter by topic tag"`
	Person string `json:"person,omitempty" jsonschema:"Filter by person mentioned"`
	Days   int    `json:"days,omitempty" jsonschema:"Only thoughts from the last N days"`
}

type thoughtStatsParams struct{}

func registerReadOnlyTools(server *mcp.Server, store *Store, cfg *Config) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search",
		Description: "Search Open Brain memories by meaning. Read-only compatibility tool for ChatGPT search/fetch access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *searchParams) (*mcp.CallToolResult, any, error) {
		results, err := runSearch(ctx, store, cfg, params.Query, 10, 0.5)
		if err != nil {
			return errorResult(err), nil, nil
		}
		type item struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
			URL   string `json:"url"`
		}
		items := make([]item, len(results))
		for i, r := range results {
			items[i] = item{ID: r.ID, Title: thoughtTitle(r.Content, r.CreatedAt), URL: thoughtURL(cfg, r.ID)}
		}
		body, _ := json.Marshal(map[string]any{"results": items})
		return textResult(string(body)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetch",
		Description: "Fetch one Open Brain thought by ID after using search.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *fetchParams) (*mcp.CallToolResult, any, error) {
		id, err := strconv.ParseInt(params.ID, 10, 64)
		if err != nil {
			return textResult(fmt.Sprintf("No thought found for ID %s.", params.ID), true), nil, nil
		}
		thought, err := store.GetByID(id)
		if err != nil {
			return errorResult(err), nil, nil
		}
		if thought == nil {
			return textResult(fmt.Sprintf("No thought found for ID %s.", params.ID), true), nil, nil
		}
		var meta map[string]any
		json.Unmarshal([]byte(thought.Metadata), &meta)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["created_at"] = thought.CreatedAt
		body, _ := json.Marshal(map[string]any{
			"id":       thought.ID,
			"title":    thoughtTitle(thought.Content, thought.CreatedAt),
			"text":     thought.Content,
			"url":      thoughtURL(cfg, thought.ID),
			"metadata": meta,
		})
		return textResult(string(body)), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_thoughts",
		Description: "Search captured thoughts by meaning.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *searchThoughtsParams) (*mcp.CallToolResult, any, error) {
		limit := params.Limit
		if limit == 0 {
			limit = 10
		}
		threshold := params.Threshold
		if threshold == 0 {
			threshold = 0.5
		}
		results, err := runSearch(ctx, store, cfg, params.Query, limit, threshold)
		if err != nil {
			return errorResult(err), nil, nil
		}
		if len(results) == 0 {
			return textResult(fmt.Sprintf("No thoughts found matching %q.", params.Query)), nil, nil
		}

		var b strings.Builder
		for i, r := range results {
			fmt.Fprintf(&b, "--- Result %d (%.1f%% match) ---\n", i+1, r.Similarity*100)
			fmt.Fprintf(&b, "Captured: %s\n", formatDate(r.CreatedAt))
			typ, _ := r.Meta["type"].(string)
			if typ == "" {
				typ = "unknown"
			}
			fmt.Fprintf(&b, "Type: %s\n", typ)
			if topics := stringSlice(r.Meta["topics"]); len(topics) > 0 {
				fmt.Fprintf(&b, "Topics: %s\n", strings.Join(topics, ", "))
			}
			if people := stringSlice(r.Meta["people"]); len(people) > 0 {
				fmt.Fprintf(&b, "People: %s\n", strings.Join(people, ", "))
			}
			if actions := stringSlice(r.Meta["action_items"]); len(actions) > 0 {
				fmt.Fprintf(&b, "Actions: %s\n", strings.Join(actions, "; "))
			}
			fmt.Fprintf(&b, "\n%s", r.Content)
			if i < len(results)-1 {
				b.WriteString("\n\n")
			}
		}
		return textResult(fmt.Sprintf("Found %d thought(s):\n\n%s", len(results), b.String())), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_thoughts",
		Description: "List recently captured thoughts with optional filters.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *listThoughtsParams) (*mcp.CallToolResult, any, error) {
		limit := params.Limit
		if limit == 0 {
			limit = 10
		}
		rows, err := store.ListThoughts(limit, params.Type, params.Topic, params.Person, params.Days)
		if err != nil {
			return errorResult(err), nil, nil
		}
		if len(rows) == 0 {
			return textResult("No thoughts found."), nil, nil
		}

		var b strings.Builder
		for i, r := range rows {
			var m map[string]any
			json.Unmarshal([]byte(r.Metadata), &m)
			typ, _ := m["type"].(string)
			if typ == "" {
				typ = "??"
			}
			suffix := typ
			if tags := strings.Join(stringSlice(m["topics"]), ", "); tags != "" {
				suffix += " - " + tags
			}
			fmt.Fprintf(&b, "%d. [%s] (%s)\n   %s", i+1, formatDate(r.CreatedAt), suffix, r.Content)
			if i < len(rows)-1 {
				b.WriteString("\n\n")
			}
		}
		return textResult(fmt.Sprintf("%d recent thought(s):\n\n%s", len(rows), b.String())), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "thought_stats",
		Description: "Get a summary of all captured thoughts.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *thoughtStatsParams) (*mcp.CallToolResult, any, error) {
		rows, err := store.ForStats()
		if err != nil {
			return errorResult(err), nil, nil
		}

		types := map[string]int{}
		topics := map[string]int{}
		people := map[string]int{}
		for _, r := range rows {
			var m map[string]any
			json.Unmarshal([]byte(r.Metadata), &m)
			if t, ok := m["type"].(string); ok && t != "" {
				types[t]++
			}
			for _, t := range stringSlice(m["topics"]) {
				topics[t]++
			}
			for _, p := range stringSlice(m["people"]) {
				people[p]++
			}
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Total thoughts: %d", len(rows)))
		if len(rows) > 0 {
			lines = append(lines, fmt.Sprintf("Date range: %s -> %s", formatDate(rows[len(rows)-1].CreatedAt), formatDate(rows[0].CreatedAt)))
		}
		lines = append(lines, "", "Types:")
		for _, kv := range topN(types, 10) {
			lines = append(lines, fmt.Sprintf("  %s: %d", kv.key, kv.count))
		}
		if len(topics) > 0 {
			lines = append(lines, "", "Top topics:")
			for _, kv := range topN(topics, 10) {
				lines = append(lines, fmt.Sprintf("  %s: %d", kv.key, kv.count))
			}
		}
		if len(people) > 0 {
			lines = append(lines, "", "People mentioned:")
			for _, kv := range topN(people, 10) {
				lines = append(lines, fmt.Sprintf("  %s: %d", kv.key, kv.count))
			}
		}
		return textResult(strings.Join(lines, "\n")), nil, nil
	})
}

type captureThoughtParams struct {
	Content string `json:"content" jsonschema:"The thought to capture"`
}

func registerCaptureThoughtTool(server *mcp.Server, store *Store, cfg *Config) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "capture_thought",
		Description: "Save a new thought to Open Brain. Generates an embedding and extracts metadata automatically.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *captureThoughtParams) (*mcp.CallToolResult, any, error) {
		var (
			embedding []float32
			meta      map[string]any
			embErr    error
			metaErr   error
		)

		done := make(chan struct{}, 2)
		go func() {
			embedding, embErr = getEmbedding(ctx, cfg, params.Content)
			done <- struct{}{}
		}()
		go func() {
			meta, metaErr = extractMetadata(ctx, cfg, params.Content)
			done <- struct{}{}
		}()
		<-done
		<-done

		if embErr != nil {
			return errorResult(embErr), nil, nil
		}
		if metaErr != nil {
			return errorResult(metaErr), nil, nil
		}

		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return errorResult(err), nil, nil
		}
		if err := store.InsertThought(params.Content, embedding, string(metaJSON)); err != nil {
			return errorResult(err), nil, nil
		}

		var b strings.Builder
		b.WriteString("Thought captured.")
		if typ, ok := meta["type"].(string); ok && typ != "" {
			fmt.Fprintf(&b, " Type: %s.", typ)
		}
		if topics := stringSlice(meta["topics"]); len(topics) > 0 {
			fmt.Fprintf(&b, " Topics: %s.", strings.Join(topics, ", "))
		}
		if people := stringSlice(meta["people"]); len(people) > 0 {
			fmt.Fprintf(&b, " People: %s.", strings.Join(people, ", "))
		}
		if actions := stringSlice(meta["action_items"]); len(actions) > 0 {
			fmt.Fprintf(&b, " Actions: %s.", strings.Join(actions, "; "))
		}
		return textResult(b.String()), nil, nil
	})
}
