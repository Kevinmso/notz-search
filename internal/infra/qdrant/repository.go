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

func (r *NoteRepository) Search(vector []float32, limit int) ([]domain.Note, error){
	return nil, nil
}

