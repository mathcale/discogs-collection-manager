package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/mathcale/discogs-collection-manager/internal/controllers"
)

type PagesSuite struct {
	suite.Suite
}

func TestPagesSuite(t *testing.T) {
	suite.Run(t, new(PagesSuite))
}

func (s *PagesSuite) TestLanding() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	(&controllers.Controller{SignupsEnabled: true}).Landing(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Discogs Collection Manager")
	s.Contains(rr.Header().Get("Content-Type"), "text/html")
}
