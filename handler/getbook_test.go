package handler_test

import (
	"avshetty1980/microservice-testing/handler"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type GetBookSuite struct {
	suite.Suite
}

func TestGetBookSuite(t *testing.T) {
	suite.Run(t, new(GetBookSuite))
}

type MockBookRetriever struct {
	mock.Mock
}

func (s *GetBookSuite) TestGetSingleBook() {

	req, _ := http.NewRequest(http.MethodGet, "/book/123456789", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("isbn", "123456789")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	resp := httptest.NewRecorder()

	br := new(MockBookRetriever)
	br.On("getBook", "123456789").Return(handler.Book{}, handler.ErrBookNotFound)

	h := handler.NewGetBook(br)
	h.ServeHTTP(resp, req)

	body, _ := io.ReadAll(resp.Body)
	// defer body.Close()

	s.Equal(http.StatusNotFound, resp.Code)
	s.JSONEq(`{"code": "001", "message": "no book with ISBN 123456789"}`, string(body))
}

func (m MockBookRetriever) GetBook(isbn string) (handler.Book, error) {
	args := m.Called(isbn)

	return args.Get(0).(handler.Book), args.Error(1)
}
