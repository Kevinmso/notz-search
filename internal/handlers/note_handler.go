package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/Kevinmso/notz-search/internal/service"
)

type NoteHandler struct {
	s *service.NoteService
}

func NewNoteHandler(s *service.NoteService) *NoteHandler {
	return &NoteHandler{s: s}
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
		http.Error(w, "failed to index note", http.StatusInternalServerError)
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
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	notes, err := h.s.Search(query, limit)
	if err != nil {
		log.Printf("failed to search notes (q=%q, limit=%d): %v", query, limit, err)
		http.Error(w, "failed to search notes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notes); err != nil {
		log.Printf("failed to encode search response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
