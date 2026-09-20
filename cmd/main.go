package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/Kevinmso/notz-search/internal/infra/embeddings"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

// newEmbeddingProvider picks the embedding provider based on the
// EMBEDDING_PROVIDER env var (defaults to "cohere").
func newEmbeddingProvider() (domain.EmbeddingProvider, error) {
	provider := os.Getenv("EMBEDDING_PROVIDER")
	if provider == "" {
		provider = "cohere"
	}

	switch provider {
	case "cohere":
		return embeddings.NewCohereProvider(os.Getenv("COHERE_API_KEY")), nil
	case "gemini":
		return embeddings.NewGeminiProvider(os.Getenv("GEMINI_API_KEY")), nil
	default:
		return nil, fmt.Errorf("unknown EMBEDDING_PROVIDER %q (expected \"cohere\" or \"gemini\")", provider)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading config from the environment")
	}

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
