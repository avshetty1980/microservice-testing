package e2e_test

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type GetSingleBookSuite struct {
	suite.Suite
}

func (s *GetSingleBookSuite) TestGetSingleBook() {

	c := http.Client{}

	res, _ := c.Get("http://localhost:8080/book/123456789")

	s.Equal(http.StatusNotFound, res.StatusCode)

	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)

	s.JSONEq(`{"code": "001", "message": "no book with ISBN 123456789"}`, string(b))
}

func (s *GetSingleBookSuite) TestGetSingleBookThatExists() {
	db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))

	db.Exec("INSERT INTO book (isbn, name, image, genre, year_published) VALUES('987654321', 'Akshay Shetty', 'testing.jpg', 'Action', 1980)")

	c := http.Client{}

	res, _ := c.Get("http://localhost:8080/book/987654321")

	s.Equal(http.StatusOK, res.StatusCode)

	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)

	expBody := `{
		"isbn": "987654321",
    "name": "Akshay Shetty",
    "image": "testing.jpg",
    "genre": "Action",
    "year_published": 1980
	}`

	s.JSONEq(expBody, string(b))
}

func TestGetSingleBookSuite(t *testing.T) {
	suite.Run(t, new(GetSingleBookSuite))
}
