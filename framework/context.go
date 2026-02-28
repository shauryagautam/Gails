package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/sessions"
)

// H is a shorthand for map[string]any, used for template data and JSON.
type H map[string]any

// HTTPError represents a typed error with an HTTP status code.
type HTTPError struct {
	Code    int
	Message string
	Errors  map[string][]string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// Context wraps http.Request and http.ResponseWriter, providing convenience methods.
type Context struct {
	Response    http.ResponseWriter
	Request     *http.Request
	app         *App
	currentUser any
	statusCode  int
	written     bool
	cacheTTL    time.Duration
}

// NewContext creates a new Context for a request.
func NewContext(w http.ResponseWriter, r *http.Request, app *App) *Context {
	return &Context{
		Response: w,
		Request:  r,
		app:      app,
	}
}

// --- URL Parameters ---

// Param returns a URL route parameter by name.
func (c *Context) Param(key string) string {
	return chi.URLParam(c.Request, key)
}

// Query returns a query string parameter by name.
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// --- Request Binding ---

var validate = validator.New()

// Bind decodes the request body (JSON or form) into v and runs validation.
// Returns an UnprocessableEntity error if validation fails.
func (c *Context) Bind(v any) error {
	var err error
	if c.IsJSON() {
		err = c.BindJSON(v)
	} else {
		err = c.BindForm(v)
	}
	if err != nil {
		return &HTTPError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	// Run struct validation
	if valErr := validate.Struct(v); valErr != nil {
		errs := make(map[string][]string)
		for _, e := range valErr.(validator.ValidationErrors) {
			field := e.Field()
			msg := fmt.Sprintf("%s is invalid (%s)", field, e.Tag())
			if e.Param() != "" {
				msg = fmt.Sprintf("%s must be %s %s", field, e.Tag(), e.Param())
			}
			errs[field] = append(errs[field], msg)
		}
		return c.UnprocessableEntity(errs)
	}

	return nil
}

// BindJSON decodes the request body as JSON.
func (c *Context) BindJSON(v any) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	defer c.Request.Body.Close()
	return json.Unmarshal(body, v)
}

// BindForm decodes form/multipart data into v using reflection.
func (c *Context) BindForm(v any) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}
	// Simple JSON roundtrip: form values → map → JSON → struct
	formMap := make(map[string]any)
	for key, values := range c.Request.Form {
		if len(values) == 1 {
			formMap[key] = values[0]
		} else {
			formMap[key] = values
		}
	}
	data, err := json.Marshal(formMap)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// --- Response ---

// JSON writes a JSON response with the given status code.
func (c *Context) JSON(status int, v any) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(status)
	c.statusCode = status
	c.written = true
	return json.NewEncoder(c.Response).Encode(v)
}

// Render renders a named template with the given data.
func (c *Context) Render(template string, data any) error {
	if c.app != nil && c.app.Renderer != nil {
		c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		c.statusCode = http.StatusOK
		c.written = true
		return c.app.Renderer.Render(c.Response, template, data)
	}
	return fmt.Errorf("renderer not initialized")
}

// Redirect sends an HTTP redirect.
func (c *Context) Redirect(url string) error {
	http.Redirect(c.Response, c.Request, url, http.StatusFound)
	c.written = true
	return nil
}

// RedirectBack redirects to the Referer header, or fallback if not present.
func (c *Context) RedirectBack(fallback string) error {
	ref := c.Request.Header.Get("Referer")
	if ref == "" {
		ref = fallback
	}
	return c.Redirect(ref)
}

// --- Sessions & Flash ---

var sessionStore sessions.Store

// InitSessionStore initializes the session store.
func InitSessionStore(secret string) {
	if secret == "" {
		secret = "gails-default-secret-change-me"
	}
	sessionStore = sessions.NewCookieStore([]byte(secret))
}

// Session returns the current session.
func (c *Context) Session() *sessions.Session {
	if sessionStore == nil {
		return nil
	}
	sess, _ := sessionStore.Get(c.Request, "gails_session")
	return sess
}

// Flash sets a flash message.
func (c *Context) Flash(key, msg string) {
	sess := c.Session()
	if sess == nil {
		return
	}
	sess.AddFlash(msg, key)
	sess.Save(c.Request, c.Response)
}

// GetFlash retrieves and clears a flash message.
func (c *Context) GetFlash(key string) string {
	sess := c.Session()
	if sess == nil {
		return ""
	}
	flashes := sess.Flashes(key)
	sess.Save(c.Request, c.Response)
	if len(flashes) > 0 {
		return fmt.Sprint(flashes[0])
	}
	return ""
}

// --- Current User ---

type contextKey string

const userContextKey contextKey = "gails_current_user"

// CurrentUser returns the current authenticated user (if set).
func (c *Context) CurrentUser() any {
	if c.currentUser != nil {
		return c.currentUser
	}
	return c.Request.Context().Value(userContextKey)
}

// SetCurrentUser sets the current user on the context.
func (c *Context) SetCurrentUser(u any) {
	c.currentUser = u
	ctx := context.WithValue(c.Request.Context(), userContextKey, u)
	c.Request = c.Request.WithContext(ctx)
}

// --- Request Info ---

// RequestID returns the request ID from the X-Request-ID header or chi middleware.
func (c *Context) RequestID() string {
	if id := middleware.GetReqID(c.Request.Context()); id != "" {
		return id
	}
	return c.Request.Header.Get("X-Request-ID")
}

// IsJSON returns true if the request Content-Type is application/json.
func (c *Context) IsJSON() bool {
	ct := c.Request.Header.Get("Content-Type")
	return strings.Contains(ct, "application/json")
}

// IsHTMX returns true if the request was made by HTMX (HX-Request header present).
func (c *Context) IsHTMX() bool {
	return c.Request.Header.Get("HX-Request") == "true"
}

// --- Error Responses ---

// BadRequest returns a 400 error.
func (c *Context) BadRequest(err error) error {
	return &HTTPError{Code: http.StatusBadRequest, Message: err.Error()}
}

// NotFound returns a 404 error.
func (c *Context) NotFound(msg string) error {
	return &HTTPError{Code: http.StatusNotFound, Message: msg}
}

// Forbidden returns a 403 error.
func (c *Context) Forbidden(msg string) error {
	return &HTTPError{Code: http.StatusForbidden, Message: msg}
}

// UnprocessableEntity returns a 422 error with field-level validation errors.
func (c *Context) UnprocessableEntity(errors map[string][]string) error {
	return &HTTPError{Code: http.StatusUnprocessableEntity, Message: "Validation failed", Errors: errors}
}

// InternalError returns a 500 error.
func (c *Context) InternalError(err error) error {
	return &HTTPError{Code: http.StatusInternalServerError, Message: err.Error()}
}

// Status writes a status code with no body.
func (c *Context) Status(code int) error {
	c.Response.WriteHeader(code)
	c.statusCode = code
	c.written = true
	return nil
}

// CacheAction marks the response to be cached for the given TTL.
func (c *Context) CacheAction(ttl time.Duration) {
	c.cacheTTL = ttl
}
