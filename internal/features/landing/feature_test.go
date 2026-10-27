package landing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/suite"

	"github.com/mathcale/discogs-collection-manager/internal/features/landing"
)

type LandingSuite struct {
	suite.Suite
}

func TestLandingSuite(t *testing.T) {
	suite.Run(t, new(LandingSuite))
}

func buildRouter(signupsEnabled bool) http.Handler {
	r := chi.NewRouter()
	landing.New(landing.Deps{SignupsEnabled: signupsEnabled}).Routes(r)
	return r
}

func (s *LandingSuite) TestLanding() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	buildRouter(true).ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Discogs Collection Manager")
	s.Contains(rr.Header().Get("Content-Type"), "text/html")
}

func (s *LandingSuite) TestLanding_RegisterLinkFollowsSignupsFlag() {
	enabled := httptest.NewRecorder()
	buildRouter(true).ServeHTTP(enabled, httptest.NewRequest(http.MethodGet, "/", nil))
	s.Contains(enabled.Body.String(), "/auth/register")

	disabled := httptest.NewRecorder()
	buildRouter(false).ServeHTTP(disabled, httptest.NewRequest(http.MethodGet, "/", nil))
	s.NotContains(disabled.Body.String(), "/auth/register")
}
