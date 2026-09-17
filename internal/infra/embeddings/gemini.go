package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Kevinmso/notz-search/internal/domain"
)

const geminiModel = "gemini-embedding-001"

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/" + geminiModel + ":embedContent"

// geminiDimensions is gemini-embedding-001's default output vector size.
const geminiDimensions = 3072

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

// GeminiProvider implements domain.EmbeddingProvider using the Gemini
// embeddings API (model gemini-embedding-001, free tier).
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewGeminiProvider builds a Gemini-backed embedding provider. apiKey is
// required; Embed returns domain.ErrEmbeddingGeneration if it's empty.
func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		apiKey:     apiKey,
		baseURL:    geminiBaseURL,
		httpClient: http.DefaultClient,
	}
}

func (p *GeminiProvider) Embed(note domain.Note) ([]float32, error) {
	if p.apiKey == "" {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
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

func (p *GeminiProvider) Dimensions() int {
	return geminiDimensions
}