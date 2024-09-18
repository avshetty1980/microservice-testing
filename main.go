package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"

	"net/http"
	"time"

	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/lib/pq"
)

type Health struct {
	Status   string   `json:"status"`
	Messages []string `json:"messages"`
}

type jsonError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Book struct {
	ISBN          string `json:"isbn"`
	Name          string `json:"name"`
	Image         string `json:"image"`
	Genre         string `json:"genre"`
	YearPublished uint16 `json:"year_published"`
}

func main() {

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error making DB connected: %s", err.Error())
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Error making DB driver: %s", err.Error())
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Error making migration engine: %s", err.Error())
	}
	migrator.Steps(2)

	r := chi.NewRouter()

	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {

		h := Health{
			Status:   "OK",
			Messages: []string{},
		}

		b, _ := json.Marshal(h)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(b))
	})

	r.Get("/book/{isbn}", func(w http.ResponseWriter, r *http.Request) {

		isbn := chi.URLParam(r, "isbn")

		book := Book{}

		row := db.QueryRow("SELECT isbn, name, image, genre, year_published FROM book WHERE isbn = $1", isbn)
		err := row.Scan(
			&book.ISBN,
			&book.Name,
			&book.Image,
			&book.Genre,
			&book.YearPublished,
		)

		if err != nil {
			bookError := jsonError{
				Code:    "001",
				Message: fmt.Sprintf("no book with ISBN %s", isbn),
			}

			body, _ := json.Marshal(bookError)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(body))
			return
		}

		body, _ := json.Marshal(book)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	s := http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	s.ListenAndServe()
}
