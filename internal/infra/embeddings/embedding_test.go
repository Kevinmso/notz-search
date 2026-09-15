package embeddings

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Kevinmso/notz-search/internal/domain"
)

// withTestServer points geminiBaseURL and httpClient at an httptest.Server
// for the duration of the test, restoring the originals afterwards.
func withTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	originalURL := geminiBaseURL
	originalClient := httpClient
	geminiBaseURL = server.URL
	httpClient = server.Client()
	t.Cleanup(func() {
		geminiBaseURL = originalURL
		httpClient = originalClient
	})
}

func withAPIKey(t *testing.T, key string) {
	t.Helper()
	original := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", key)
	t.Cleanup(func() {
		os.Setenv("GEMINI_API_KEY", original)
	})
}

func TestNoteToEmbedding_MissingAPIKey(t *testing.T) {
	withAPIKey(t, "")

	_, err := NoteToEmbedding(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}

func TestNoteToEmbedding_Success(t *testing.T) {
	withAPIKey(t, "test-key")

	var receivedReq embedRequest
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
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
	emb, err := NoteToEmbedding(note)
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

func TestNoteToEmbedding_APIError(t *testing.T) {
	withAPIKey(t, "test-key")

	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid API key"}}`))
	})

	_, err := NoteToEmbedding(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}

func TestNoteToEmbedding_InvalidJSONResponse(t *testing.T) {
	withAPIKey(t, "test-key")

	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	})

	_, err := NoteToEmbedding(domain.Note{Title: "t", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected error to wrap domain.ErrEmbeddingGeneration, got %v", err)
	}
}
