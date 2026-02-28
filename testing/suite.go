package testing

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/shaurya/gails/framework"
	"gorm.io/gorm"
)

// Suite provides test utilities for Gails applications.
type Suite struct {
	DB      *gorm.DB
	App     *framework.App
	Server  *httptest.Server
	Factory *Factory
	Assert  *Assertions
	t       *testing.T
}

// NewSuite creates a new test suite. Call in TestMain or individual tests.
func NewSuite(t *testing.T) *Suite {
	os.Setenv("APP_ENV", "test")

	app := framework.New()

	s := &Suite{
		App:     app,
		DB:      app.DB,
		Factory: NewFactory(),
		Assert:  &Assertions{t: t},
		t:       t,
	}

	// Start test HTTP server
	s.Server = httptest.NewServer(app.Router.Mux)

	return s
}

// Close cleans up the suite.
func (s *Suite) Close() {
	if s.Server != nil {
		s.Server.Close()
	}
}

// GET sends a GET request.
func (s *Suite) GET(path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", path, nil)
	rr := httptest.NewRecorder()
	s.App.Router.Mux.ServeHTTP(rr, req)
	return rr
}

// POST sends a POST request with a JSON body.
func (s *Suite) POST(path string, body framework.H) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.App.Router.Mux.ServeHTTP(rr, req)
	return rr
}

// PUT sends a PUT request with a JSON body.
func (s *Suite) PUT(path string, body framework.H) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.App.Router.Mux.ServeHTTP(rr, req)
	return rr
}

// DELETE sends a DELETE request.
func (s *Suite) DELETE(path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("DELETE", path, nil)
	rr := httptest.NewRecorder()
	s.App.Router.Mux.ServeHTTP(rr, req)
	return rr
}

// GETWithAuth sends a GET request with a Bearer token.
func (s *Suite) GETWithAuth(path, token string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	s.App.Router.Mux.ServeHTTP(rr, req)
	return rr
}

// --- Assertions ---

// Assertions provides test assertion helpers.
type Assertions struct {
	t *testing.T
}

// Equal asserts that two values are equal.
func (a *Assertions) Equal(expected, actual any) {
	a.t.Helper()
	if expected != actual {
		a.t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// JSONContains asserts the response body contains a string.
func (a *Assertions) JSONContains(res *httptest.ResponseRecorder, substr string) {
	a.t.Helper()
	body := res.Body.String()
	if !contains(body, substr) {
		a.t.Errorf("Expected response body to contain %q, got %q", substr, body)
	}
}

// HTMLContains asserts the response body contains HTML content.
func (a *Assertions) HTMLContains(res *httptest.ResponseRecorder, substr string) {
	a.t.Helper()
	body := res.Body.String()
	if !contains(body, substr) {
		a.t.Errorf("Expected HTML to contain %q", substr)
	}
}

// Redirects asserts the response redirects to a URL.
func (a *Assertions) Redirects(res *httptest.ResponseRecorder, url string) {
	a.t.Helper()
	if res.Code != http.StatusFound && res.Code != http.StatusMovedPermanently {
		a.t.Errorf("Expected redirect status, got %d", res.Code)
	}
	location := res.Header().Get("Location")
	if location != url {
		a.t.Errorf("Expected redirect to %q, got %q", url, location)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
