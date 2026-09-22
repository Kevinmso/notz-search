package service

import (
	"errors"
	"testing"

	"github.com/Kevinmso/notz-search/internal/domain"
)

type fakeEmbedder struct {
	vector      []float32
	err         error
	calls       []domain.Note
	queryVector []float32
	queryErr    error
	queryCalls  []string
}

func (f *fakeEmbedder) Embed(note domain.Note) ([]float32, error) {
	f.calls = append(f.calls, note)
	return f.vector, f.err
}

func (f *fakeEmbedder) EmbedQuery(query string) ([]float32, error) {
	f.queryCalls = append(f.queryCalls, query)
	return f.queryVector, f.queryErr
}

func (f *fakeEmbedder) Dimensions() int { return len(f.vector) }

type fakeRepo struct {
	upsertErr    error
	searchResult []domain.Note
	searchErr    error
	deleteErr    error

	upsertCalls   int
	upsertedNote  domain.Note
	upsertedVec   []float32
	searchCalls   int
	searchedVec   []float32
	searchedLimit int
	deleteCalls   int
	deletedID     string
}

func (f *fakeRepo) Upsert(note domain.Note, vector []float32) error {
	f.upsertCalls++
	f.upsertedNote = note
	f.upsertedVec = vector
	return f.upsertErr
}

func (f *fakeRepo) Search(vector []float32, limit int) ([]domain.Note, error) {
	f.searchCalls++
	f.searchedVec = vector
	f.searchedLimit = limit
	return f.searchResult, f.searchErr
}

func (f *fakeRepo) Delete(id string) error {
	f.deleteCalls++
	f.deletedID = id
	return f.deleteErr
}

func equalVectors(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestIndexNote_EmbedsThenUpserts(t *testing.T) {
	embedder := &fakeEmbedder{vector: []float32{0.1, 0.2}}
	repo := &fakeRepo{}
	svc := NewNoteService(repo, embedder)

	note := domain.Note{ID: "id-1", Title: "t", Text: "x"}
	if err := svc.IndexNote(note); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embedder.calls) != 1 || embedder.calls[0].ID != note.ID {
		t.Fatalf("embedder should be called once with the note, got %+v", embedder.calls)
	}
	if repo.upsertCalls != 1 {
		t.Fatalf("Upsert calls = %d, want 1", repo.upsertCalls)
	}
	if repo.upsertedNote.ID != note.ID {
		t.Errorf("upserted note ID = %q, want %q", repo.upsertedNote.ID, note.ID)
	}
	if !equalVectors(repo.upsertedVec, embedder.vector) {
		t.Errorf("upserted vector = %v, want the embedder's vector %v", repo.upsertedVec, embedder.vector)
	}
}

func TestIndexNote_EmbedErrorSkipsUpsert(t *testing.T) {
	embedder := &fakeEmbedder{err: domain.ErrEmbeddingGeneration}
	repo := &fakeRepo{}
	svc := NewNoteService(repo, embedder)

	err := svc.IndexNote(domain.Note{ID: "id-1", Text: "x"})
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected domain.ErrEmbeddingGeneration, got %v", err)
	}
	if repo.upsertCalls != 0 {
		t.Errorf("Upsert must not run when embedding fails, got %d calls", repo.upsertCalls)
	}
}

func TestIndexNote_UpsertErrorIsPropagated(t *testing.T) {
	embedder := &fakeEmbedder{vector: []float32{0.1}}
	repo := &fakeRepo{upsertErr: domain.ErrUpsert}
	svc := NewNoteService(repo, embedder)

	err := svc.IndexNote(domain.Note{ID: "id-1", Text: "x"})
	if !errors.Is(err, domain.ErrUpsert) {
		t.Fatalf("expected domain.ErrUpsert, got %v", err)
	}
}

func TestSearch_EmbedsQueryThenSearches(t *testing.T) {
	want := []domain.Note{{ID: "a"}, {ID: "b"}}
	embedder := &fakeEmbedder{vector: []float32{9, 9}, queryVector: []float32{0.3, 0.4}}
	repo := &fakeRepo{searchResult: want}
	svc := NewNoteService(repo, embedder)

	got, err := svc.Search("minha pergunta", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embedder.queryCalls) != 1 || embedder.queryCalls[0] != "minha pergunta" {
		t.Fatalf("the raw query should go through EmbedQuery once, got %+v", embedder.queryCalls)
	}
	if len(embedder.calls) != 0 {
		t.Errorf("Search must not embed the query as a document, got %d Embed calls", len(embedder.calls))
	}
	if repo.searchCalls != 1 || repo.searchedLimit != 5 {
		t.Errorf("Search called %d times with limit %d, want 1 call with limit 5", repo.searchCalls, repo.searchedLimit)
	}
	if !equalVectors(repo.searchedVec, embedder.queryVector) {
		t.Errorf("searched vector = %v, want the query vector %v", repo.searchedVec, embedder.queryVector)
	}
	if len(got) != len(want) || got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("results = %+v, want %+v", got, want)
	}
}

func TestSearch_EmbedErrorSkipsSearch(t *testing.T) {
	embedder := &fakeEmbedder{queryErr: domain.ErrEmbeddingGeneration}
	repo := &fakeRepo{}
	svc := NewNoteService(repo, embedder)

	_, err := svc.Search("q", 10)
	if !errors.Is(err, domain.ErrEmbeddingGeneration) {
		t.Fatalf("expected domain.ErrEmbeddingGeneration, got %v", err)
	}
	if repo.searchCalls != 0 {
		t.Errorf("Search must not run when embedding fails, got %d calls", repo.searchCalls)
	}
}

func TestSearch_RepoErrorIsPropagated(t *testing.T) {
	embedder := &fakeEmbedder{queryVector: []float32{0.1}}
	repo := &fakeRepo{searchErr: domain.ErrSearch}
	svc := NewNoteService(repo, embedder)

	_, err := svc.Search("q", 10)
	if !errors.Is(err, domain.ErrSearch) {
		t.Fatalf("expected domain.ErrSearch, got %v", err)
	}
}

func TestDeleteNote_DelegatesToRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewNoteService(repo, &fakeEmbedder{})

	if err := svc.DeleteNote("id-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.deleteCalls != 1 || repo.deletedID != "id-1" {
		t.Errorf("Delete called %d time(s) with id %q, want 1 call with id-1", repo.deleteCalls, repo.deletedID)
	}
}

func TestDeleteNote_RepoErrorIsPropagated(t *testing.T) {
	repo := &fakeRepo{deleteErr: domain.ErrDelete}
	svc := NewNoteService(repo, &fakeEmbedder{})

	err := svc.DeleteNote("id-1")
	if !errors.Is(err, domain.ErrDelete) {
		t.Fatalf("expected domain.ErrDelete, got %v", err)
	}
}
