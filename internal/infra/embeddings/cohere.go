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

const (
	cohereModel   = "embed-multilingual-v3.0"
	cohereBaseURL = "https://api.cohere.com/v2/embed"
	// cohereDimensions is embed-multilingual-v3.0's output vector size.
	cohereDimensions = 1024

	cohereInputDocument = "search_document"
	cohereInputQuery    = "search_query"
)

type cohereEmbedRequest struct {
	Model          string   `json:"model"`
	Texts          []string `json:"texts"`
	InputType      string   `json:"input_type"`
	EmbeddingTypes []string `json:"embedding_types"`
}

type cohereEmbedResponse struct {
	Embeddings struct {
		Float [][]float32 `json:"float"`
	} `json:"embeddings"`
}

// CohereProvider implements domain.EmbeddingProvider using the Cohere
// embeddings API (model embed-multilingual-v3.0, free trial key).
type CohereProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewCohereProvider builds a Cohere-backed embedding provider. apiKey is
// required; Embed returns domain.ErrEmbeddingGeneration if it's empty.
func NewCohereProvider(apiKey string) *CohereProvider {
	return &CohereProvider{
		apiKey:     apiKey,
		baseURL:    cohereBaseURL,
		httpClient: http.DefaultClient,
	}
}

// Embed embeds a note as a document to be searched against.
func (p *CohereProvider) Embed(note domain.Note) ([]float32, error) {
	return p.embed(note.Title+"\n\n"+note.Text, cohereInputDocument)
}

// EmbedQuery embeds a search query. Cohere's v3 models are asymmetric, so
// queries must use the search_query input type to match search_document ones.
func (p *CohereProvider) EmbedQuery(query string) ([]float32, error) {
	return p.embed(query, cohereInputQuery)
}

func (p *CohereProvider) embed(text, inputType string) ([]float32, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("%w: COHERE_API_KEY is not set", domain.ErrEmbeddingGeneration)
	}

	reqBody, err := json.Marshal(cohereEmbedRequest{
		Model:          cohereModel,
		Texts:          []string{text},
		InputType:      inputType,
		EmbeddingTypes: []string{"float"},
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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: cohere api returned %d: %s", domain.ErrEmbeddingGeneration, resp.StatusCode, body)
	}

	var out cohereEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}

	if len(out.Embeddings.Float) == 0 {
		return nil, fmt.Errorf("%w: cohere api returned no embeddings", domain.ErrEmbeddingGeneration)
	}

	return out.Embeddings.Float[0], nil
}

func (p *CohereProvider) Dimensions() int {
	return cohereDimensions
}
