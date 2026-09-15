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

type NoteRepository interface {
	Upsert(note Note) error
	Search(vector []float32, limit int) ([]Note, error)
}