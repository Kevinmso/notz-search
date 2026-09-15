package domain

import "errors"

var (
	// ErrEmbeddingGeneration indicates the text-to-vector conversion failed.
	ErrEmbeddingGeneration = errors.New("failed to generate embedding")

	// ErrUpsert indicates the vector store failed to persist a note.
	ErrUpsert = errors.New("failed to upsert note")

	// ErrSearch indicates the vector store failed to run a similarity search.
	ErrSearch = errors.New("failed to search notes")
	
	// ErrInvalidPayload indicates a note read back from the vector store has a
	// malformed payload (e.g. an unparseable timestamp).
	ErrInvalidPayload = errors.New("invalid note payload")
)
