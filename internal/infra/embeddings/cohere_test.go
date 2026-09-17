package embeddings

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kevinmso/notz-search/internal/domain"
)

// newTestCohereProvider builds a CohereProvider pointed at an httptest.Server
// instead of the real Cohere API.
func newTestCohereProvider(t *testing.T, apiKey string, handler http.HandlerFunc) *CohereProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &CohereProvider{
		apiKey:     apiKey,
		baseURL:    server.URL,
		httpClient: server.Client(),
	}
}

func TestCohereProvider_Embed_MissingAPIKey(t *testing.T) {
	provider := NewCohereProvider("")

	_, err := provider.Embed(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}

func TestCohereProvider_Embed_Success(t *testing.T) {
	var receivedReq cohereEmbedRequest
	provider := newTestCohereProvider(t, "test-key", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-key")
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cohereEmbedResponse{
			Embeddings: struct {
				Float [][]float32 `json:"float"`
			}{Float: [][]float32{{0.1, 0.2, 0.3}}},
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
	if len(receivedReq.Texts) != 1 || receivedReq.Texts[0] != wantText {
		t.Errorf("sent texts = %+v, want a single text %q", receivedReq.Texts, wantText)
	}
}

func TestCohereProvider_Embed_APIError(t *testing.T) {
	provider := newTestCohereProvider(t, "test-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"invalid api token"}`))
	})

	_, err := provider.Embed(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}

func TestCohereProvider_Embed_InvalidJSONResponse(t *testing.T) {
	provider := newTestCohereProvider(t, "test-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	})

	_, err := provider.Embed(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}

func TestCohereProvider_Embed_NoEmbeddingsReturned(t *testing.T) {
	provider := newTestCohereProvider(t, "test-key", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cohereEmbedResponse{})
	})

	_, err := provider.Embed(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}
