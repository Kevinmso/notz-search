package domain

// EmbeddingProvider converts text into vector embeddings.
// Implementations live in internal/infra/embeddings, one per provider
// (Gemini, OpenAI, Cohere, ...), keeping the provider choice out of the
// domain and repository layers.
type EmbeddingProvider interface {
	// Embed vectorizes a note so it can be stored and later retrieved.
	Embed(note Note) ([]float32, error)
	// EmbedQuery vectorizes a search query. It's separate from Embed because
	// retrieval models are asymmetric: queries and documents must be embedded
	// with different modes to land close to each other in the vector space.
	EmbedQuery(query string) ([]float32, error)
	// Dimensions returns the length of the vectors Embed produces. The
	// vector store needs it upfront to create a collection with a matching
	// fixed size.
	Dimensions() int
}
