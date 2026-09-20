package domain

import "time"

type Note struct {
	ID        string
	Title     string
	Text      string
	LinksTo   []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NoteRepository persists and retrieves notes by vector similarity. It deals
// only in already-computed vectors — generating them is EmbeddingProvider's
// job, kept separate so storage and embedding can vary independently.
type NoteRepository interface {
	Upsert(note Note, vector []float32) error
	Search(vector []float32, limit int) ([]Note, error)
}
