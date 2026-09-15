package qdrant

import (
	"context"
	"fmt"
	"time"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/Kevinmso/notz-search/internal/infra/embeddings"
	"github.com/qdrant/go-client/qdrant"
)

const COLLECTION_NAME = "notes"

type NoteRepository struct {
	client *qdrant.Client // qdrant real client lib
}

func NewNoteRepository(client *qdrant.Client) *NoteRepository {
	return &NoteRepository{client:client}
}

func (r *NoteRepository) Upsert(note domain.Note) error {
	emb, err := embeddings.NoteToEmbedding(note)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	linksTo := make([]interface{}, len(note.LinksTo))
	for i, link := range note.LinksTo {
		linksTo[i] = link
	}

	payload := qdrant.NewValueMap(map[string]any{
		"title":      note.Title,
		"text":       note.Text,
		"links_to":   linksTo,
		"created_at": note.CreatedAt.Format(time.RFC3339),
		"updated_at": note.UpdatedAt.Format(time.RFC3339),
	})

	// create Qdrant points before upsert
	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDUUID(note.ID),
			Vectors: qdrant.NewVectors(emb...),
			Payload: payload,
		},
	}

	_, err = r.client.Upsert(context.Background(), &qdrant.UpsertPoints{
		CollectionName: COLLECTION_NAME,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

func (r *NoteRepository) Search(vector []float32, limit int) ([]domain.Note, error) {
	scoredPoints, err := r.client.Query(context.Background(), &qdrant.QueryPoints{
		CollectionName: COLLECTION_NAME,
		Query:          qdrant.NewQuery(vector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search points: %w", err)
	}

	notes := make([]domain.Note, len(scoredPoints))
	for i, sp := range scoredPoints {
		payload := sp.Payload

		linksTo := make([]string, 0)
		if list := payload["links_to"].GetListValue(); list != nil {
			for _, v := range list.Values {
				linksTo = append(linksTo, v.GetStringValue())
			}
		}

		createdAt, _ := time.Parse(time.RFC3339, payload["created_at"].GetStringValue())
		updatedAt, _ := time.Parse(time.RFC3339, payload["updated_at"].GetStringValue())

		notes[i] = domain.Note{
			ID:        sp.Id.GetUuid(),
			Title:     payload["title"].GetStringValue(),
			Text:      payload["text"].GetStringValue(),
			LinksTo:   linksTo,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
	}

	return notes, nil
}

