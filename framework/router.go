package framework

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RouteInfo stores metadata about a registered route for the route inspector.
type RouteInfo struct {
	Method  string
	Path    string
	Handler string
}

// Router wraps chi.Mux with Rails-like conventions.
type Router struct {
	Mux    *chi.Mux
	app    *App
	routes []RouteInfo
	prefix string
}

// NewRouter creates a new Router.
func NewRouter() *Router {
	return &Router{
		Mux:    chi.NewRouter(),
		routes: make([]RouteInfo, 0),
	}
}

func (r *Router) addRoute(method, path, handler string) {
	fullPath := r.prefix + path
	r.routes = append(r.routes, RouteInfo{Method: method, Path: fullPath, Handler: handler})
}

// Use adds middleware to the router.
func (r *Router) Use(mw func(http.Handler) http.Handler) {
	r.Mux.Use(mw)
}

// GET registers a GET route.
func (r *Router) GET(path string, handler Action) {
	r.addRoute("GET", path, actionName(handler))
	r.Mux.Get(r.prefix+path, ActionHandler(handler, r.app))
}

// POST registers a POST route.
func (r *Router) POST(path string, handler Action) {
	r.addRoute("POST", path, actionName(handler))
	r.Mux.Post(r.prefix+path, ActionHandler(handler, r.app))
}

// PUT registers a PUT route.
func (r *Router) PUT(path string, handler Action) {
	r.addRoute("PUT", path, actionName(handler))
	r.Mux.Put(r.prefix+path, ActionHandler(handler, r.app))
}

// PATCH registers a PATCH route.
func (r *Router) PATCH(path string, handler Action) {
	r.addRoute("PATCH", path, actionName(handler))
	r.Mux.Patch(r.prefix+path, ActionHandler(handler, r.app))
}

// DELETE registers a DELETE route.
func (r *Router) DELETE(path string, handler Action) {
	r.addRoute("DELETE", path, actionName(handler))
	r.Mux.Delete(r.prefix+path, ActionHandler(handler, r.app))
}

// Mount mounts a sub-handler at a prefix.
func (r *Router) Mount(path string, handler http.Handler) {
	r.addRoute("*", path+"/*", "Mounted Handler")
	r.Mux.Mount(path, handler)
}

// WebSocket registers a WebSocket route.
func (r *Router) WebSocket(path string, handler http.HandlerFunc) {
	r.addRoute("WS", path, "WebSocket")
	r.Mux.Get(path, handler)
}

// Namespace creates a sub-group with a prefix.
func (r *Router) Namespace(prefix string, fn func(r *Router)) {
	subRouter := &Router{
		Mux:    chi.NewRouter(),
		app:    r.app,
		routes: r.routes,
		prefix: r.prefix + prefix,
	}
	fn(subRouter)
	r.routes = subRouter.routes
	r.Mux.Mount(prefix, subRouter.Mux)
}

// Resources registers RESTful routes for a controller.
// Controllers only need to implement the actions they handle — missing actions return 404.
func (r *Router) Resources(name string, controller any, fn ...func(r *Router)) {
	prefix := "/" + name
	controllerName := fmt.Sprintf("%T", controller)
	// Clean up pointer prefix
	if strings.HasPrefix(controllerName, "*") {
		controllerName = controllerName[1:]
	}
	// Extract just the type name without package
	parts := strings.Split(controllerName, ".")
	if len(parts) > 1 {
		controllerName = parts[len(parts)-1]
	}

	r.Mux.Route(prefix, func(router chi.Router) {
		app := r.app

		// Index: GET /resources
		if c, ok := controller.(interface{ Index(*Context) error }); ok {
			r.addRoute("GET", prefix, controllerName+"#Index")
			router.Get("/", ActionHandler(c.Index, app))
		}
		// Create: POST /resources
		if c, ok := controller.(interface{ Create(*Context) error }); ok {
			r.addRoute("POST", prefix, controllerName+"#Create")
			router.Post("/", ActionHandler(c.Create, app))
		}
		// New: GET /resources/new
		if c, ok := controller.(interface{ New(*Context) error }); ok {
			r.addRoute("GET", prefix+"/new", controllerName+"#New")
			router.Get("/new", ActionHandler(c.New, app))
		}
		// Show: GET /resources/{id}
		if c, ok := controller.(interface{ Show(*Context) error }); ok {
			r.addRoute("GET", prefix+"/{id}", controllerName+"#Show")
			router.Get("/{id}", ActionHandler(c.Show, app))
		}
		// Edit: GET /resources/{id}/edit
		if c, ok := controller.(interface{ Edit(*Context) error }); ok {
			r.addRoute("GET", prefix+"/{id}/edit", controllerName+"#Edit")
			router.Get("/{id}/edit", ActionHandler(c.Edit, app))
		}
		// Update: PUT /resources/{id}
		if c, ok := controller.(interface{ Update(*Context) error }); ok {
			r.addRoute("PUT", prefix+"/{id}", controllerName+"#Update")
			router.Put("/{id}", ActionHandler(c.Update, app))
			r.addRoute("PATCH", prefix+"/{id}", controllerName+"#Update")
			router.Patch("/{id}", ActionHandler(c.Update, app))
		}
		// Destroy: DELETE /resources/{id}
		if c, ok := controller.(interface{ Destroy(*Context) error }); ok {
			r.addRoute("DELETE", prefix+"/{id}", controllerName+"#Destroy")
			router.Delete("/{id}", ActionHandler(c.Destroy, app))
		}

		// Nested resources
		if len(fn) > 0 {
			nestedRouter := &Router{
				Mux:    chi.NewRouter(),
				app:    app,
				routes: r.routes,
				prefix: prefix + "/{" + singleName(name) + "_id}",
			}
			fn[0](nestedRouter)
			r.routes = nestedRouter.routes
			router.Mount("/{"+singleName(name)+"_id}", nestedRouter.Mux)
		}
	})
}

// Inspect returns all registered routes as a formatted table.
func (r *Router) Inspect() string {
	var sb strings.Builder

	// Header
	sb.WriteString("┌────────────┬────────────────────────────────┬────────────────────────────────┐\n")
	sb.WriteString("│ Method     │ Path                           │ Handler                        │\n")
	sb.WriteString("├────────────┼────────────────────────────────┼────────────────────────────────┤\n")

	for _, route := range r.routes {
		sb.WriteString(fmt.Sprintf("│ %-10s │ %-30s │ %-30s │\n", route.Method, route.Path, route.Handler))
	}

	sb.WriteString("└────────────┴────────────────────────────────┴────────────────────────────────┘\n")
	sb.WriteString(fmt.Sprintf("Total: %d routes\n", len(r.routes)))

	return sb.String()
}

// GetRoutes returns all registered route info.
func (r *Router) GetRoutes() []RouteInfo {
	return r.routes
}

// singleName converts a plural resource name to singular (basic singularization).
func singleName(name string) string {
	if strings.HasSuffix(name, "ies") {
		return name[:len(name)-3] + "y"
	}
	if strings.HasSuffix(name, "ses") || strings.HasSuffix(name, "xes") {
		return name[:len(name)-2]
	}
	if strings.HasSuffix(name, "s") {
		return name[:len(name)-1]
	}
	return name
}

// actionName returns a string name for a handler function (for route table display).
func actionName(_ Action) string {
	return "Action"
}
