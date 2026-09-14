package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/health", func (w http.ResponseWriter, r *http.Request)  {
		w.Write([]byte("ok"))
	})

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil{
		log.Fatal(err)
	}
}