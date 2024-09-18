package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Book struct {
	ISBN          string `json:"isbn"`
	Name          string `json:"name"`
	Image         string `json:"image"`
	Genre         string `json:"genre"`
	YearPublished uint16 `json:"year_published"`
}

type jsonError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	ErrBookNotFound = errors.New("book not found")
)

type BookRetriever interface {
	GetBook(isbn string) (Book, error)
}

type GetBookHandler struct {
}

func (g GetBookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isbn := chi.URLParam(r, "isbn")

	bookError := jsonError{
		Code:    "001",
		Message: fmt.Sprintf("no book with ISBN %s", isbn),
	}

	body, _ := json.Marshal(bookError)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(body))

}

func NewGetBook(br BookRetriever) GetBookHandler {
	return GetBookHandler{}
}
