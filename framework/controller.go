package framework

import (
	"net/http"

	"go.uber.org/zap"
)

// Controller is the base type for all controllers. Embed it in your controllers.
type Controller struct{}

// Action is a controller action handler signature.
type Action func(ctx *Context) error

// ResourceController defines the interface for RESTful resource controllers.
// Controllers only implement the actions they need — unimplemented actions return 404.
type ResourceController interface {
	Index(ctx *Context) error
	Show(ctx *Context) error
	New(ctx *Context) error
	Create(ctx *Context) error
	Edit(ctx *Context) error
	Update(ctx *Context) error
	Destroy(ctx *Context) error
}

// ActionHandler converts an Action into an http.HandlerFunc.
// It creates a Context, calls the action, and handles errors with correct HTTP status codes.
func ActionHandler(action Action, app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r, app)

		// Panic recovery per-request
		defer func() {
			if err := recover(); err != nil {
				if app != nil && app.Config != nil && app.Config.App.Env == "development" {
					DevErrorHandler(w, r, err)
				} else {
					ProdErrorHandler(w, r, err)
				}
			}
		}()

		err := action(ctx)
		if err != nil {
			handleActionError(ctx, err)
		}
	}
}

// handleActionError maps errors to the correct HTTP status codes.
func handleActionError(ctx *Context, err error) {
	if ctx.written {
		return // Response already sent
	}

	if httpErr, ok := err.(*HTTPError); ok {
		if ctx.IsJSON() || httpErr.Errors != nil {
			response := H{"error": httpErr.Message}
			if httpErr.Errors != nil {
				response = H{"errors": httpErr.Errors}
			}
			ctx.JSON(httpErr.Code, response)
		} else {
			http.Error(ctx.Response, httpErr.Message, httpErr.Code)
		}
		return
	}

	// Default: 500 Internal Server Error
	if Log != nil {
		Log.Error("Unhandled controller error", zap.Error(err))
	}
	if ctx.IsJSON() {
		ctx.JSON(http.StatusInternalServerError, H{"error": "Internal Server Error"})
	} else {
		http.Error(ctx.Response, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Wrap is a convenience alias for ActionHandler without an app reference.
func Wrap(action Action) http.HandlerFunc {
	return ActionHandler(action, nil)
}
