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
