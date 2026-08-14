package main

import (
	"math"
	"testing"
)

func TestEncodeDecodeEmbeddingRoundtrip(t *testing.T) {
	in := []float32{0.1, -0.5, 3.25, 0}
	out := decodeEmbedding(encodeEmbedding(in))
	if len(out) != len(in) {
		t.Fatalf("length = %d, want %d", len(out), len(in))
	}
	for i := range in {
		if out[i] != in[i] {
			t.Errorf("index %d: got %v, want %v", i, out[i], in[i])
		}
	}
}

func TestCosineSimilarityIdentical(t *testing.T) {
	v := []float32{1, 2, 3}
	got := cosineSimilarity(v, v)
	if math.Abs(got-1.0) > 1e-6 {
		t.Errorf("cosineSimilarity(v, v) = %v, want ~1.0", got)
	}
}

func TestCosineSimilarityOrthogonal(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}
	got := cosineSimilarity(a, b)
	if math.Abs(got) > 1e-6 {
		t.Errorf("cosineSimilarity(orthogonal) = %v, want ~0", got)
	}
}

func TestCosineSimilarityZeroNorm(t *testing.T) {
	a := []float32{0, 0}
	b := []float32{1, 2}
	got := cosineSimilarity(a, b)
	if got != 0 {
		t.Errorf("cosineSimilarity(zero vector) = %v, want 0", got)
	}
}

func TestStoreInsertAndGetByID(t *testing.T) {
	store, err := openStore(":memory:")
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	if err := store.InsertThought("hello world", []float32{0.1, 0.2}, `{"type":"observation"}`); err != nil {
		t.Fatalf("InsertThought: %v", err)
	}

	row, err := store.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if row == nil {
		t.Fatal("GetByID(1) = nil, want row")
	}
	if row.Content != "hello world" {
		t.Errorf("Content = %q", row.Content)
	}

	missing, err := store.GetByID(999)
	if err != nil {
		t.Fatalf("GetByID(999): %v", err)
	}
	if missing != nil {
		t.Error("GetByID(999) = non-nil, want nil for missing row")
	}
}

func TestStoreDeleteThought(t *testing.T) {
	store, err := openStore(":memory:")
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	if err := store.InsertThought("doomed", nil, `{"type":"observation"}`); err != nil {
		t.Fatalf("InsertThought: %v", err)
	}

	deleted, err := store.DeleteThought(1)
	if err != nil {
		t.Fatalf("DeleteThought: %v", err)
	}
	if !deleted {
		t.Error("DeleteThought(1) = false, want true")
	}

	row, err := store.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if row != nil {
		t.Error("GetByID(1) = non-nil after delete")
	}

	deleted, err = store.DeleteThought(1)
	if err != nil {
		t.Fatalf("DeleteThought(missing): %v", err)
	}
	if deleted {
		t.Error("DeleteThought(missing) = true, want false")
	}
}

func TestStoreListThoughtsFilters(t *testing.T) {
	store, err := openStore(":memory:")
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	store.InsertThought("a task", nil, `{"type":"task","topics":["work"],"people":["Alice"]}`)
	store.InsertThought("an idea", nil, `{"type":"idea","topics":["fun"],"people":[]}`)

	rows, err := store.ListThoughts(10, "task", "", "", 0)
	if err != nil {
		t.Fatalf("ListThoughts: %v", err)
	}
	if len(rows) != 1 || rows[0].Content != "a task" {
		t.Errorf("ListThoughts(type=task) = %+v", rows)
	}

	rows, err = store.ListThoughts(10, "", "", "Alice", 0)
	if err != nil {
		t.Fatalf("ListThoughts: %v", err)
	}
	if len(rows) != 1 || rows[0].Content != "a task" {
		t.Errorf("ListThoughts(person=Alice) = %+v", rows)
	}
}
