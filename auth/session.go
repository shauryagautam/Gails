package auth

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/shaurya/gails/framework"
)

var store sessions.Store

// InitSession initializes the session store.
func InitSession(secret string) {
	if secret == "" {
		secret = "change_me_in_production"
	}
	store = sessions.NewCookieStore([]byte(secret))
}

// SessionMiddleware loads the user session on each request.
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// Login logs in a user by setting their ID in the session.
// It regenerates the session ID to prevent session fixation.
func Login(w http.ResponseWriter, r *http.Request, userID uint) error {
	if store == nil {
		InitSession("")
	}
	session, _ := store.Get(r, "gails_session")
	// Regenerate session ID to prevent fixation
	session.Options.MaxAge = 86400
	session.Values["user_id"] = userID
	return session.Save(r, w)
}

// Logout logs out the current user by clearing the session.
func Logout(w http.ResponseWriter, r *http.Request) error {
	if store == nil {
		return nil
	}
	session, _ := store.Get(r, "gails_session")
	session.Values["user_id"] = nil
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// Required wraps an Action to require authentication.
func Required(next framework.Action) framework.Action {
	return func(ctx *framework.Context) error {
		sess := ctx.Session()
		if sess == nil {
			return ctx.Forbidden("Authentication required")
		}
		uid := sess.Values["user_id"]
		if uid == nil {
			return ctx.Forbidden("Authentication required")
		}
		return next(ctx)
	}
}

// RequireRole wraps an Action to require a specific role.
func RequireRole(role string, next framework.Action) framework.Action {
	return func(ctx *framework.Context) error {
		sess := ctx.Session()
		if sess == nil {
			return ctx.Forbidden("Authentication required")
		}
		uid := sess.Values["user_id"]
		if uid == nil {
			return ctx.Forbidden("Authentication required")
		}
		userRole, _ := sess.Values["role"].(string)
		if userRole != role {
			return ctx.Forbidden("Insufficient permissions")
		}
		return next(ctx)
	}
}
