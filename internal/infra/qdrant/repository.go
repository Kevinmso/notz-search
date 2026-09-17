package qdrant

import (
	"context"
	"fmt"
	"time"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/qdrant/go-client/qdrant"
)

const COLLECTION_NAME = "notes"

type NoteRepository struct {
	client   *qdrant.Client // qdrant real client lib
	embedder domain.EmbeddingProvider
}

func NewNoteRepository(client *qdrant.Client, embedder domain.EmbeddingProvider) *NoteRepository {
	return &NoteRepository{client: client, embedder: embedder}
}

// EnsureCollection creates the notes collection if it doesn't already exist,
// sized for vectors of vectorSize dimensions compared by cosine distance.
// It's a no-op if the collection is already there.
func (r *NoteRepository) EnsureCollection(ctx context.Context, vectorSize int) error {
	exists, err := r.client.CollectionExists(ctx, COLLECTION_NAME)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrEnsureCollection, err)
	}
	if exists {
		return nil
	}

	err = r.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: COLLECTION_NAME,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(vectorSize),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrEnsureCollection, err)
	}

	return nil
}

// noteToPayload converts a Note's fields (all but ID and the embedding vector,
// which are carried separately in a Qdrant point) into a Qdrant payload map.
func noteToPayload(note domain.Note) map[string]*qdrant.Value {
	linksTo := make([]interface{}, len(note.LinksTo))
	for i, link := range note.LinksTo {
		linksTo[i] = link
	}

	return qdrant.NewValueMap(map[string]any{
		"title":      note.Title,
		"text":       note.Text,
		"links_to":   linksTo,
		"created_at": note.CreatedAt.Format(time.RFC3339),
		"updated_at": note.UpdatedAt.Format(time.RFC3339),
	})
}

// scoredPointToNote converts a Qdrant search result back into a Note.
// It returns domain.ErrInvalidPayload if a timestamp field can't be parsed.
func scoredPointToNote(sp *qdrant.ScoredPoint) (domain.Note, error) {
	payload := sp.Payload

	linksTo := make([]string, 0)
	if list := payload["links_to"].GetListValue(); list != nil {
		for _, v := range list.Values {
			linksTo = append(linksTo, v.GetStringValue())
		}
	}

	createdAt, err := time.Parse(time.RFC3339, payload["created_at"].GetStringValue())
	if err != nil {
		return domain.Note{}, fmt.Errorf("%w: created_at: %w", domain.ErrInvalidPayload, err)
	}

	updatedAt, err := time.Parse(time.RFC3339, payload["updated_at"].GetStringValue())
	if err != nil {
		return domain.Note{}, fmt.Errorf("%w: updated_at: %w", domain.ErrInvalidPayload, err)
	}

	return domain.Note{
		ID:        sp.Id.GetUuid(),
		Title:     payload["title"].GetStringValue(),
		Text:      payload["text"].GetStringValue(),
		LinksTo:   linksTo,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (r *NoteRepository) Upsert(note domain.Note) error {
	emb, err := r.embedder.Embed(note)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrEmbeddingGeneration, err)
	}

	// create Qdrant points before upsert
	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDUUID(note.ID),
			Vectors: qdrant.NewVectors(emb...),
			Payload: noteToPayload(note),
		},
	}

	_, err = r.client.Upsert(context.Background(), &qdrant.UpsertPoints{
		CollectionName: COLLECTION_NAME,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrUpsert, err)
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
		return nil, fmt.Errorf("%w: %w", domain.ErrSearch, err)
	}

	notes := make([]domain.Note, len(scoredPoints))
	for i, sp := range scoredPoints {
		note, err := scoredPointToNote(sp)
		if err != nil {
			return nil, err
		}
		notes[i] = note
	}

	return notes, nil
}