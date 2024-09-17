package e2e_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type EndToEndSuite struct {
	suite.Suite
}

func (s *EndToEndSuite) TestHealthCheck() {

	c := http.Client{}

	res, _ := c.Get("http://localhost:8080/status")

	s.Equal(http.StatusOK, res.StatusCode)

	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)

	s.JSONEq(`{"status": "OK", "messages": []}`, string(b))
}

func TestEndToEndSuite(t *testing.T) {
	suite.Run(t, new(EndToEndSuite))
}
