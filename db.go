package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS thoughts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	content TEXT NOT NULL,
	embedding BLOB,
	metadata TEXT DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_thoughts_created_at ON thoughts (created_at DESC);
`

type ThoughtRow struct {
	ID        int64
	Content   string
	Embedding []byte
	Metadata  string
	CreatedAt string
}

type Store struct {
	db               *sql.DB
	insertStmt       *sql.Stmt
	byIDStmt         *sql.Stmt
	allForSearchStmt *sql.Stmt
	forStatsStmt     *sql.Stmt
}

func openStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL"); err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, err
	}

	insertStmt, err := db.Prepare("INSERT INTO thoughts (content, embedding, metadata) VALUES (?, ?, ?)")
	if err != nil {
		return nil, err
	}
	byIDStmt, err := db.Prepare("SELECT id, content, metadata, created_at FROM thoughts WHERE id = ? LIMIT 1")
	if err != nil {
		return nil, err
	}
	allForSearchStmt, err := db.Prepare("SELECT id, content, embedding, metadata, created_at FROM thoughts WHERE embedding IS NOT NULL")
	if err != nil {
		return nil, err
	}
	forStatsStmt, err := db.Prepare("SELECT metadata, created_at FROM thoughts ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}

	return &Store{
		db:               db,
		insertStmt:       insertStmt,
		byIDStmt:         byIDStmt,
		allForSearchStmt: allForSearchStmt,
		forStatsStmt:     forStatsStmt,
	}, nil
}

func (s *Store) InsertThought(content string, embedding []float32, metadata string) error {
	var blob []byte
	if embedding != nil {
		blob = encodeEmbedding(embedding)
	}
	_, err := s.insertStmt.Exec(content, blob, metadata)
	return err
}

func (s *Store) GetByID(id int64) (*ThoughtRow, error) {
	var t ThoughtRow
	err := s.byIDStmt.QueryRow(id).Scan(&t.ID, &t.Content, &t.Metadata, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) AllForSearch() ([]ThoughtRow, error) {
	rows, err := s.allForSearchStmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ThoughtRow
	for rows.Next() {
		var t ThoughtRow
		if err := rows.Scan(&t.ID, &t.Content, &t.Embedding, &t.Metadata, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ForStats() ([]ThoughtRow, error) {
	rows, err := s.forStatsStmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ThoughtRow
	for rows.Next() {
		var t ThoughtRow
		if err := rows.Scan(&t.Metadata, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ListThoughts(limit int, typ, topic, person string, days int) ([]ThoughtRow, error) {
	query := "SELECT content, metadata, created_at FROM thoughts WHERE 1=1"
	var params []any

	if typ != "" {
		query += " AND json_extract(metadata, '$.type') = ?"
		params = append(params, typ)
	}
	if topic != "" {
		query += " AND EXISTS (SELECT 1 FROM json_each(json_extract(metadata, '$.topics')) WHERE value = ?)"
		params = append(params, topic)
	}
	if person != "" {
		query += " AND EXISTS (SELECT 1 FROM json_each(json_extract(metadata, '$.people')) WHERE value = ?)"
		params = append(params, person)
	}
	if days > 0 {
		query += " AND created_at >= datetime('now', ?)"
		params = append(params, fmt.Sprintf("-%d days", days))
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	params = append(params, limit)

	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ThoughtRow
	for rows.Next() {
		var t ThoughtRow
		if err := rows.Scan(&t.Content, &t.Metadata, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func encodeEmbedding(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func decodeEmbedding(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func cosineSimilarity(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
