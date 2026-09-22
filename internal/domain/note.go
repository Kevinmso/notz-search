package domain

import "time"

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	LinksTo   []string  `json:"links_to"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NoteRepository persists and retrieves notes by vector similarity. It deals
// only in already-computed vectors — generating them is EmbeddingProvider's
// job, kept separate so storage and embedding can vary independently.
type NoteRepository interface {
	Upsert(note Note, vector []float32) error
	Search(vector []float32, limit int) ([]Note, error)
}
