package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// DataChunk methods (for RAG)
func (s *SQLiteStore) createDataChunk(chunk *DataChunk) error {
	embeddingBytes, err := json.Marshal(chunk.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}
	chunk.EmbeddingJSON = string(embeddingBytes)

	stmt, err := s.db.Prepare("INSERT INTO data_chunks (content, embedding_json) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare data_chunk insert: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(chunk.Content, chunk.EmbeddingJSON)
	if err != nil {
		return fmt.Errorf("failed to execute data_chunk insert: %w", err)
	}
	chunk.ID, _ = res.LastInsertId()
	return nil
}

func (s *SQLiteStore) GetAllDataChunks() ([]DataChunk, error) {
	rows, err := s.db.Query("SELECT id, content, embedding_json FROM data_chunks")
	if err != nil {
		return nil, fmt.Errorf("failed to query data_chunks: %w", err)
	}
	defer rows.Close()

	var chunks []DataChunk
	for rows.Next() {
		var chunk DataChunk
		var embeddingJSON string // Read as string from DB
		if err := rows.Scan(&chunk.ID, &chunk.Content, &embeddingJSON); err != nil {
			return nil, fmt.Errorf("failed to scan data_chunk row: %w", err)
		}
		// Ensure embeddingJSON is not empty before trying to unmarshal
		if embeddingJSON != "" {
			if err := json.Unmarshal([]byte(embeddingJSON), &chunk.Embedding); err != nil {
				log.Printf("Warning: failed to unmarshal embedding for chunk %d (content: %.50s...): %v. Embedding will be empty.", chunk.ID, chunk.Content, err)
				// Chunk will have an empty embedding, which might affect similarity search.
				// Consider if this should be a fatal error for the chunk or if it's acceptable.
				chunk.Embedding = nil // Explicitly set to nil if unmarshal fails
			}
		} else {
			log.Printf("Warning: empty embedding_json for chunk ID %d. Embedding will be empty.", chunk.ID)
			chunk.Embedding = nil // Ensure it's nil if the DB field was empty/NULL
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (s *SQLiteStore) ClearDataChunks() error {
	_, err := s.db.Exec("DELETE FROM data_chunks")
	if err != nil {
		return fmt.Errorf("failed to delete data_chunks: %w", err)
	}
	_, err = s.db.Exec("DELETE FROM sqlite_sequence WHERE name='data_chunks'")
	if err != nil && !strings.Contains(err.Error(), "no such table") {
		log.Printf("Warning: could not reset sequence for data_chunks: %v", err)
	}
	return nil
}

// IngestDataFromFile reads data.md, extracts text, generates embeddings, and stores them.
func (s *SQLiteStore) IngestDataFromFile(filePath string, embedder func(string) ([]float32, error)) (int, error) {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read data file %s: %w", filePath, err)
	}
	fileContent := string(contentBytes)
	lines := strings.Split(fileContent, "\n")

	var rawChunks []string
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue // Skip empty lines
		}

		// Skip table header and separator
		if i == 0 && strings.Contains(trimmedLine, "|") && (strings.Contains(strings.ToLower(trimmedLine), "text") || strings.Contains(strings.ToLower(trimmedLine), "content")) {
			log.Printf("Skipping table header: %s", trimmedLine)
			continue
		}
		if i == 1 && strings.Contains(trimmedLine, "|") && strings.Contains(trimmedLine, "---") {
			log.Printf("Skipping table separator: %s", trimmedLine)
			continue
		}

		// Basic parsing for a single-column Markdown table row: | some content |
		if strings.HasPrefix(trimmedLine, "|") && strings.HasSuffix(trimmedLine, "|") {
			parts := strings.Split(trimmedLine, "|")
			// Expect 3 parts: "" (before first |), " content ", "" (after last |)
			// Or for | text | header, parts would be ["", " text ", ""]
			if len(parts) >= 3 { // At least | content |
				// The actual content is the second element after splitting by '|', then trim spaces.
				// Example: "| some content |" -> parts are ["", " some content ", ""]
				// Example: "|text|" -> parts are ["", "text", ""]
				// We take parts[1] which is " some content " and trim it.
				cellContent := strings.TrimSpace(parts[1])
				if cellContent != "" {
					rawChunks = append(rawChunks, cellContent)
				} else {
					log.Printf("Skipping row with empty cell content: %s", trimmedLine)
				}
			} else {
				log.Printf("Skipping malformed table row (not enough '|'): %s", trimmedLine)
			}
		} else {
			// If it's not a table row after the header, skip it.
			if i > 1 { // Only log if we're past the typical header/separator lines
				log.Printf("Skipping line not matching table row format: %s", trimmedLine)
			}
		}
	}

	if len(rawChunks) == 0 {
		log.Println("No chunks generated from data file. Ensure it's a Markdown table with a 'text' column and content.")
		return 0, nil // Or an error if this is unexpected
	}

	log.Printf("Generated %d raw chunks from table. Now embedding (this may take a while)...", len(rawChunks))

	if err := s.ClearDataChunks(); err != nil {
		return 0, fmt.Errorf("failed to clear existing data chunks: %w", err)
	}

	count := 0

	ticker := time.NewTicker(40 * time.Millisecond) // delay to not hit rate limit (1500/min)
	defer ticker.Stop()

	for i, rawChunk := range rawChunks {
		<-ticker.C

		embedding, err := embedder(rawChunk)
		if err != nil {
			log.Printf("Failed to generate embedding for chunk %d (\"%.50s...\"): %v. Skipping.", i+1, rawChunk, err)
			continue
		}

		chunk := DataChunk{
			Content:   rawChunk,
			Embedding: embedding,
		}
		if err := s.createDataChunk(&chunk); err != nil {
			log.Printf("Failed to store data chunk %d: %v. Skipping.", i+1, err)
			continue
		}
		count++
		if count%10 == 0 || count == len(rawChunks) {
			log.Printf("Ingested %d/%d chunks...", count, len(rawChunks))
		}
	}
	log.Printf("Successfully ingested %d chunks.", count)
	return count, nil
}
