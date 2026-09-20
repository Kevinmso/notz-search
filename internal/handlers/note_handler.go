package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/Kevinmso/notz-search/internal/service"
	"github.com/google/uuid"
)

type NoteHandler struct {
	s *service.NoteService
}

func NewNoteHandler(s *service.NoteService) *NoteHandler {
	return &NoteHandler{s: s}
}

// serviceErrorStatus maps a service-layer error to the HTTP status that best
// describes it. Failures of the embedding provider or the vector store are
// upstream problems (502); anything else, including a malformed payload read
// back from our own index, is an internal error (500).
func serviceErrorStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrEmbeddingGeneration),
		errors.Is(err, domain.ErrUpsert),
		errors.Is(err, domain.ErrSearch):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

type indexNoteRequest struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Text    string   `json:"text"`
	LinksTo []string `json:"links_to"`
}

func (h *NoteHandler) IndexNotes(w http.ResponseWriter, r *http.Request) {
	var req indexNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(req.ID); err != nil {
		http.Error(w, "id must be a valid UUID", http.StatusBadRequest)
		return
	}

	note := domain.Note{
		ID:        req.ID,
		Title:     req.Title,
		Text:      req.Text,
		LinksTo:   req.LinksTo,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.s.IndexNote(note); err != nil {
		log.Printf("failed to index note %q: %v", note.ID, err)
		http.Error(w, "failed to index note", serviceErrorStatus(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *NoteHandler) SearchNotes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing required query param: q", http.StatusBadRequest)
		return
	}

	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	notes, err := h.s.Search(query, limit)
	if err != nil {
		log.Printf("failed to search notes (q=%q, limit=%d): %v", query, limit, err)
		http.Error(w, "failed to search notes", serviceErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notes); err != nil {
		log.Printf("failed to encode search response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
