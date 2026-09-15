package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Kevinmso/notz-search/internal/domain"
)

const geminiModel = "gemini-embedding-001"

// geminiBaseURL and httpClient are package-level vars (not consts) so tests
// can point them at an httptest.Server instead of the real Gemini API.
var (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/" + geminiModel + ":embedContent"
	httpClient    = http.DefaultClient
)

type embedRequest struct {
	Model   string       `json:"model"`
	Content embedContent `json:"content"`
}

type embedContent struct {
	Parts []embedPart `json:"parts"`
}

type embedPart struct {
	Text string `json:"text"`
}

type embedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// NoteToEmbedding converts a Note's text into a vector using the Gemini
// embeddings API (model gemini-embedding-001, free tier). Requires the
// GEMINI_API_KEY environment variable to be set.
func NoteToEmbedding(note domain.Note) ([]float32, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("%w: GEMINI_API_KEY is not set", domain.ErrEmbeddingGeneration)
	}

	reqBody, err := json.Marshal(embedRequest{
		Model: "models/" + geminiModel,
		Content: embedContent{
			Parts: []embedPart{{Text: note.Title + "\n\n" + note.Text}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, geminiBaseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: gemini api returned %d: %s", domain.ErrEmbeddingGeneration, resp.StatusCode, body)
	}

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}

	return out.Embedding.Values, nil
}
