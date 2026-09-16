package domain

// EmbeddingProvider converts a Note's text into a vector embedding.
// Implementations live in internal/infra/embeddings, one per provider
// (Gemini, OpenAI, Cohere, ...), keeping the provider choice out of the
// domain and repository layers.
type EmbeddingProvider interface {
	Embed(note Note) ([]float32, error)
}