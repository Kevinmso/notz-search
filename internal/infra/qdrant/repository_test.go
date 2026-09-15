package qdrant

import (
	"errors"
	"testing"
	"time"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/qdrant/go-client/qdrant"
)

func TestNoteToPayload(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 1, 3, 11, 0, 0, 0, time.UTC)

	note := domain.Note{
		ID:        "note-1",
		Title:     "Título",
		Text:      "Conteúdo da nota",
		LinksTo:   []string{"nota-a", "nota-b"},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	payload := noteToPayload(note)

	if got := payload["title"].GetStringValue(); got != note.Title {
		t.Errorf("title = %q, want %q", got, note.Title)
	}
	if got := payload["text"].GetStringValue(); got != note.Text {
		t.Errorf("text = %q, want %q", got, note.Text)
	}
	if got := payload["created_at"].GetStringValue(); got != createdAt.Format(time.RFC3339) {
		t.Errorf("created_at = %q, want %q", got, createdAt.Format(time.RFC3339))
	}
	if got := payload["updated_at"].GetStringValue(); got != updatedAt.Format(time.RFC3339) {
		t.Errorf("updated_at = %q, want %q", got, updatedAt.Format(time.RFC3339))
	}

	list := payload["links_to"].GetListValue()
	if list == nil {
		t.Fatal("links_to should be a list value")
	}
	if len(list.Values) != len(note.LinksTo) {
		t.Fatalf("links_to has %d items, want %d", len(list.Values), len(note.LinksTo))
	}
	for i, v := range list.Values {
		if got := v.GetStringValue(); got != note.LinksTo[i] {
			t.Errorf("links_to[%d] = %q, want %q", i, got, note.LinksTo[i])
		}
	}
}

func TestNoteToPayload_EmptyLinks(t *testing.T) {
	note := domain.Note{
		ID:        "note-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LinksTo:   nil,
	}

	payload := noteToPayload(note)

	list := payload["links_to"].GetListValue()
	if list == nil {
		t.Fatal("links_to should be a list value even when empty")
	}
	if len(list.Values) != 0 {
		t.Errorf("links_to should be empty, got %d items", len(list.Values))
	}
}

func buildScoredPoint(t *testing.T, id string, extra map[string]any) *qdrant.ScoredPoint {
	t.Helper()
	return &qdrant.ScoredPoint{
		Id:      qdrant.NewIDUUID(id),
		Payload: qdrant.NewValueMap(extra),
	}
}

func TestScoredPointToNote(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 1, 3, 11, 0, 0, 0, time.UTC)

	sp := buildScoredPoint(t, "note-1", map[string]any{
		"title":      "Título",
		"text":       "Conteúdo da nota",
		"links_to":   []any{"nota-a", "nota-b"},
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	})

	note, err := scoredPointToNote(sp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if note.ID != "note-1" {
		t.Errorf("ID = %q, want %q", note.ID, "note-1")
	}
	if note.Title != "Título" {
		t.Errorf("Title = %q, want %q", note.Title, "Título")
	}
	if note.Text != "Conteúdo da nota" {
		t.Errorf("Text = %q, want %q", note.Text, "Conteúdo da nota")
	}
	if !note.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v", note.CreatedAt, createdAt)
	}
	if !note.UpdatedAt.Equal(updatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", note.UpdatedAt, updatedAt)
	}

	wantLinks := []string{"nota-a", "nota-b"}
	if len(note.LinksTo) != len(wantLinks) {
		t.Fatalf("LinksTo = %v, want %v", note.LinksTo, wantLinks)
	}
	for i, link := range wantLinks {
		if note.LinksTo[i] != link {
			t.Errorf("LinksTo[%d] = %q, want %q", i, note.LinksTo[i], link)
		}
	}
}

func TestScoredPointToNote_InvalidCreatedAt(t *testing.T) {
	sp := buildScoredPoint(t, "note-1", map[string]any{
		"title":      "Título",
		"text":       "Conteúdo",
		"links_to":   []any{},
		"created_at": "not-a-timestamp",
		"updated_at": time.Now().Format(time.RFC3339),
	})

	_, err := scoredPointToNote(sp)
	if !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected error to wrap domain.ErrInvalidPayload, got %v", err)
	}
}

func TestScoredPointToNote_InvalidUpdatedAt(t *testing.T) {
	sp := buildScoredPoint(t, "note-1", map[string]any{
		"title":      "Título",
		"text":       "Conteúdo",
		"links_to":   []any{},
		"created_at": time.Now().Format(time.RFC3339),
		"updated_at": "not-a-timestamp",
	})

	_, err := scoredPointToNote(sp)
	if !errors.Is(err, domain.ErrInvalidPayload) {
		t.Fatalf("expected error to wrap domain.ErrInvalidPayload, got %v", err)
	}
}

func TestScoredPointToNote_NoLinks(t *testing.T) {
	sp := buildScoredPoint(t, "note-1", map[string]any{
		"title":      "Título",
		"text":       "Conteúdo",
		"created_at": time.Now().Format(time.RFC3339),
		"updated_at": time.Now().Format(time.RFC3339),
	})

	note, err := scoredPointToNote(sp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(note.LinksTo) != 0 {
		t.Errorf("LinksTo = %v, want empty", note.LinksTo)
	}
}
