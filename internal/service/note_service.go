package service

import "github.com/Kevinmso/notz-search/internal/domain"

// NoteService orchestrates embedding generation and note storage/retrieval,
// keeping that coordination out of both the domain and the infra adapters.
type NoteService struct {
	repo     domain.NoteRepository
	embedder domain.EmbeddingProvider
}

func NewNoteService(repo domain.NoteRepository, embedder domain.EmbeddingProvider) *NoteService {
	return &NoteService{repo: repo, embedder: embedder}
}

// IndexNote generates an embedding for note and persists it.
func (s *NoteService) IndexNote(note domain.Note) error {
	vector, err := s.embedder.Embed(note)
	if err != nil {
		return err
	}
	return s.repo.Upsert(note, vector)
}

// Search embeds query and returns the most similar notes, up to limit.
func (s *NoteService) Search(query string, limit int) ([]domain.Note, error) {
	vector, err := s.embedder.EmbedQuery(query)
	if err != nil {
		return nil, err
	}
	return s.repo.Search(vector, limit)
}
