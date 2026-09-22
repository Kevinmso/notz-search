package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Kevinmso/notz-search/internal/domain"
	"github.com/Kevinmso/notz-search/internal/handlers"
	"github.com/Kevinmso/notz-search/internal/infra/embeddings"
	"github.com/Kevinmso/notz-search/internal/infra/qdrant"
	"github.com/Kevinmso/notz-search/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	qdrantclient "github.com/qdrant/go-client/qdrant"
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

	qdrantUseTLS, _ := strconv.ParseBool(os.Getenv("QDRANT_USE_TLS"))

	qdrantClient, err := qdrantclient.NewClient(&qdrantclient.Config{
		Host:   os.Getenv("QDRANT_HOST"),
		Port:   6334,
		APIKey: os.Getenv("QDRANT_API_KEY"),
		UseTLS: qdrantUseTLS,
	})
	if err != nil {
		log.Fatalf("failed to connect to qdrant: %v", err)
	}

	embedder, err := newEmbeddingProvider()
	if err != nil {
		log.Fatal(err)
	}

	noteRepository := qdrant.NewNoteRepository(qdrantClient)
	if err := noteRepository.EnsureCollection(context.Background(), embedder.Dimensions()); err != nil {
		log.Fatalf("failed to ensure qdrant collection: %v", err)
	}

	noteService := service.NewNoteService(noteRepository, embedder)
	noteHandler := handlers.NewNoteHandler(noteService)

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Post("/notes/index", noteHandler.IndexNotes)
	r.Get("/search", noteHandler.SearchNotes)
	r.Delete("/notes/{id}", noteHandler.DeleteNote)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
