package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/Kevinmso/notz-search/internal/service"
)

const validUUID = "550e8400-e29b-41d4-a716-446655440000"

type fakeEmbedder struct {
	vector []float32
	err    error
}

func (f *fakeEmbedder) Embed(domain.Note) ([]float32, error) { return f.vector, f.err }
func (f *fakeEmbedder) EmbedQuery(string) ([]float32, error) { return f.vector, f.err }
func (f *fakeEmbedder) Dimensions() int                      { return len(f.vector) }

type fakeRepo struct {
	upsertErr    error
	searchResult []domain.Note
	searchErr    error

	upsertCalls   int
	upsertedNote  domain.Note
	searchCalls   int
	searchedLimit int
}

func (f *fakeRepo) Upsert(note domain.Note, _ []float32) error {
	f.upsertCalls++
	f.upsertedNote = note
	return f.upsertErr
}

func (f *fakeRepo) Search(_ []float32, limit int) ([]domain.Note, error) {
	f.searchCalls++
	f.searchedLimit = limit
	return f.searchResult, f.searchErr
}

func newHandler(repo *fakeRepo, embedder *fakeEmbedder) *NoteHandler {
	return NewNoteHandler(service.NewNoteService(repo, embedder))
}

func newHealthyHandler() (*NoteHandler, *fakeRepo) {
	repo := &fakeRepo{}
	return newHandler(repo, &fakeEmbedder{vector: []float32{0.1, 0.2}}), repo
}

func postIndex(h *NoteHandler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/notes/index", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.IndexNotes(rec, req)
	return rec
}

func getSearch(h *NoteHandler, params url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/search?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	h.SearchNotes(rec, req)
	return rec
}

func indexBody(id, text string) string {
	return fmt.Sprintf(`{"id":%q,"title":"Título","text":%q,"links_to":["a","b"]}`, id, text)
}

func TestIndexNotes_Success(t *testing.T) {
	h, repo := newHealthyHandler()

	rec := postIndex(h, indexBody(validUUID, "conteúdo"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if repo.upsertCalls != 1 {
		t.Fatalf("Upsert calls = %d, want 1", repo.upsertCalls)
	}

	got := repo.upsertedNote
	if got.ID != validUUID || got.Title != "Título" || got.Text != "conteúdo" {
		t.Errorf("upserted note = %+v, fields don't match the request", got)
	}
	if len(got.LinksTo) != 2 || got.LinksTo[0] != "a" || got.LinksTo[1] != "b" {
		t.Errorf("LinksTo = %v, want [a b]", got.LinksTo)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("CreatedAt and UpdatedAt must be set by the server")
	}
}

func TestIndexNotes_BadRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{not json`},
		{"missing text", indexBody(validUUID, "")},
		{"id is not a UUID", indexBody("nota-1", "conteúdo")},
		{"empty id", indexBody("", "conteúdo")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newHealthyHandler()

			rec := postIndex(h, tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if repo.upsertCalls != 0 {
				t.Errorf("Upsert must not run for an invalid request, got %d calls", repo.upsertCalls)
			}
		})
	}
}

func TestIndexNotes_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		embedder *fakeEmbedder
		repo     *fakeRepo
		want     int
	}{
		{
			name:     "embedding provider failure is a bad gateway",
			embedder: &fakeEmbedder{err: fmt.Errorf("%w: cohere down", domain.ErrEmbeddingGeneration)},
			repo:     &fakeRepo{},
			want:     http.StatusBadGateway,
		},
		{
			name:     "vector store failure is a bad gateway",
			embedder: &fakeEmbedder{vector: []float32{0.1}},
			repo:     &fakeRepo{upsertErr: fmt.Errorf("%w: qdrant down", domain.ErrUpsert)},
			want:     http.StatusBadGateway,
		},
		{
			name:     "unrecognized failure is an internal error",
			embedder: &fakeEmbedder{vector: []float32{0.1}},
			repo:     &fakeRepo{upsertErr: errors.New("something unexpected")},
			want:     http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postIndex(newHandler(tt.repo, tt.embedder), indexBody(validUUID, "conteúdo"))

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestSearchNotes_Success(t *testing.T) {
	h, repo := newHealthyHandler()
	repo.searchResult = []domain.Note{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}}

	rec := getSearch(h, url.Values{"q": {"pergunta"}, "limit": {"3"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if repo.searchedLimit != 3 {
		t.Errorf("limit passed to the repository = %d, want 3", repo.searchedLimit)
	}

	var got []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("response ids = %+v, want [a b]", got)
	}
}

func TestSearchNotes_DefaultLimit(t *testing.T) {
	h, repo := newHealthyHandler()

	rec := getSearch(h, url.Values{"q": {"pergunta"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if repo.searchedLimit != 10 {
		t.Errorf("default limit = %d, want 10", repo.searchedLimit)
	}
}

func TestSearchNotes_BadRequests(t *testing.T) {
	tests := []struct {
		name   string
		params url.Values
	}{
		{"missing q", url.Values{}},
		{"empty q", url.Values{"q": {""}}},
		{"non-numeric limit", url.Values{"q": {"x"}, "limit": {"abc"}}},
		{"zero limit", url.Values{"q": {"x"}, "limit": {"0"}}},
		{"negative limit", url.Values{"q": {"x"}, "limit": {"-1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newHealthyHandler()

			rec := getSearch(h, tt.params)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if repo.searchCalls != 0 {
				t.Errorf("Search must not run for an invalid request, got %d calls", repo.searchCalls)
			}
		})
	}
}

func TestSearchNotes_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		embedder *fakeEmbedder
		repo     *fakeRepo
		want     int
	}{
		{
			name:     "embedding provider failure is a bad gateway",
			embedder: &fakeEmbedder{err: fmt.Errorf("%w: cohere down", domain.ErrEmbeddingGeneration)},
			repo:     &fakeRepo{},
			want:     http.StatusBadGateway,
		},
		{
			name:     "vector store failure is a bad gateway",
			embedder: &fakeEmbedder{vector: []float32{0.1}},
			repo:     &fakeRepo{searchErr: fmt.Errorf("%w: qdrant down", domain.ErrSearch)},
			want:     http.StatusBadGateway,
		},
		{
			name:     "corrupt payload in our own index is an internal error",
			embedder: &fakeEmbedder{vector: []float32{0.1}},
			repo:     &fakeRepo{searchErr: fmt.Errorf("%w: created_at", domain.ErrInvalidPayload)},
			want:     http.StatusInternalServerError,
		},
		{
			name:     "unrecognized failure is an internal error",
			embedder: &fakeEmbedder{vector: []float32{0.1}},
			repo:     &fakeRepo{searchErr: errors.New("something unexpected")},
			want:     http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := getSearch(newHandler(tt.repo, tt.embedder), url.Values{"q": {"pergunta"}})

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
