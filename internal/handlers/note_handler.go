package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Kevinmso/notz-search/internal/service"
)

type NoteHandler struct {
	s *service.NoteService
}

func NewNoteHandler(s *service.NoteService) *NoteHandler {
	return &NoteHandler{s: s}
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
		http.Error(w, "failed to search notes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notes); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}