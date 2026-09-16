package embeddings

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kevinmso/notz-search/internal/domain"
)

// newTestProvider builds a GeminiProvider pointed at an httptest.Server
// instead of the real Gemini API.
func newTestProvider(t *testing.T, apiKey string, handler http.HandlerFunc) *GeminiProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &GeminiProvider{
		apiKey:     apiKey,
		baseURL:    server.URL,
		httpClient: server.Client(),
	}
}

func TestGeminiProvider_Embed_MissingAPIKey(t *testing.T) {
	provider := NewGeminiProvider("")

	_, err := provider.Embed(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}

func TestGeminiProvider_Embed_Success(t *testing.T) {
	var receivedReq embedRequest
	provider := newTestProvider(t, "test-key", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Errorf("x-goog-api-key header = %q, want %q", got, "test-key")
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(embedResponse{
			Embedding: struct {
				Values []float32 `json:"values"`
			}{Values: []float32{0.1, 0.2, 0.3}},
		})
	})

	note := domain.Note{Title: "Título", Text: "Conteúdo da nota"}
	emb, err := provider.Embed(note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []float32{0.1, 0.2, 0.3}
	if len(emb) != len(want) {
		t.Fatalf("embedding = %v, want %v", emb, want)
	}
	for i := range want {
		if emb[i] != want[i] {
			t.Errorf("embedding[%d] = %v, want %v", i, emb[i], want[i])
		}
	}

	wantText := note.Title + "\n\n" + note.Text
	if len(receivedReq.Content.Parts) != 1 || receivedReq.Content.Parts[0].Text != wantText {
		t.Errorf("sent text = %+v, want a single part with %q", receivedReq.Content.Parts, wantText)
	}
}

func TestGeminiProvider_Embed_APIError(t *testing.T) {
	provider := newTestProvider(t, "test-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid API key"}}`))
	})

	_, err := provider.Embed(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}

func TestGeminiProvider_Embed_InvalidJSONResponse(t *testing.T) {
	provider := newTestProvider(t, "test-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	})

	_, err := provider.Embed(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}