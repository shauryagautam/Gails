package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/shaurya/gails/auth"
	"github.com/shaurya/gails/cache"
	"github.com/shaurya/gails/config"
	"github.com/shaurya/gails/framework"
	"github.com/shaurya/gails/framework/i18n"
	"github.com/shaurya/gails/mailer"
	"github.com/shaurya/gails/orm"
	"github.com/shaurya/gails/plugins/admin"
	"github.com/shaurya/gails/plugins/healthcheck"
	"github.com/shaurya/gails/queue"
	"github.com/shaurya/gails/queue/dashboard"
	"github.com/shaurya/gails/websocket"
)

// ──────────────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────────────

type User struct {
	orm.Model
	Name     string `gorm:"not null" validate:"required"`
	Email    string `gorm:"uniqueIndex;not null" validate:"required,email"`
	Role     string `gorm:"default:user"`
	Password string `gorm:"not null" json:"-"`
	Posts    []Post `gorm:"foreignKey:UserID"`
}

type Post struct {
	orm.Model
	Title    string `gorm:"not null" validate:"required"`
	Body     string `gorm:"type:text"`
	UserID   uint   `gorm:"not null;index"`
	User     User
	Comments []Comment `gorm:"foreignKey:PostID"`
}

type Comment struct {
	orm.Model
	Body   string `gorm:"type:text;not null" validate:"required"`
	PostID uint   `gorm:"not null;index"`
	Post   Post
	UserID uint `gorm:"not null;index"`
	User   User
}

// ──────────────────────────────────────────────────────────────────────────────
// Controllers
// ──────────────────────────────────────────────────────────────────────────────

type UsersController struct{ framework.Controller }

func (c *UsersController) Index(ctx *framework.Context) error {
	return ctx.JSON(http.StatusOK, framework.H{"action": "users#index"})
}

func (c *UsersController) Show(ctx *framework.Context) error {
	id := ctx.Param("id")
	return ctx.JSON(http.StatusOK, framework.H{"action": "users#show", "id": id})
}

func (c *UsersController) Create(ctx *framework.Context) error {
	var input struct {
		Name     string `json:"name" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}

	if err := ctx.Bind(&input); err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, framework.H{
		"user": framework.H{"name": input.Name, "email": input.Email},
	})
}

type PostsController struct{ framework.Controller }

func (c *PostsController) Index(ctx *framework.Context) error {
	return ctx.JSON(http.StatusOK, framework.H{"action": "posts#index"})
}

func (c *PostsController) Show(ctx *framework.Context) error {
	return ctx.JSON(http.StatusOK, framework.H{"action": "posts#show", "id": ctx.Param("id")})
}

func (c *PostsController) Create(ctx *framework.Context) error {
	var input struct {
		Title string `json:"title" validate:"required"`
		Body  string `json:"body"`
	}
	if err := ctx.Bind(&input); err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, framework.H{"post": input})
}

func (c *PostsController) Update(ctx *framework.Context) error {
	return ctx.JSON(http.StatusOK, framework.H{"action": "posts#update", "id": ctx.Param("id")})
}

func (c *PostsController) Destroy(ctx *framework.Context) error {
	return ctx.JSON(http.StatusOK, framework.H{"action": "posts#destroy", "id": ctx.Param("id")})
}

type CommentsController struct{ framework.Controller }

func (c *CommentsController) Index(ctx *framework.Context) error {
	return ctx.JSON(http.StatusOK, framework.H{
		"action":  "comments#index",
		"post_id": ctx.Param("post_id"),
	})
}

func (c *CommentsController) Create(ctx *framework.Context) error {
	return ctx.JSON(http.StatusCreated, framework.H{
		"action":  "comments#create",
		"post_id": ctx.Param("post_id"),
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Mailers
// ──────────────────────────────────────────────────────────────────────────────

type WelcomeMailer struct {
	mailer.Mailer
}

func (m WelcomeMailer) WelcomeEmail(name, email string) *mailer.Email {
	return m.NewEmail().
		To(email).
		Subject("Welcome to Gails, " + name + "!").
		HTMLBody(fmt.Sprintf("<h1>Welcome, %s!</h1><p>Your account has been created.</p>", name))
}

// ──────────────────────────────────────────────────────────────────────────────
// WebSocket
// ──────────────────────────────────────────────────────────────────────────────

type ChatChannel struct{}

func (ch *ChatChannel) OnConnect(ctx *websocket.WSContext) error {
	ctx.JoinRoom("general")
	return nil
}

func (ch *ChatChannel) OnMessage(ctx *websocket.WSContext, msg []byte) error {
	ctx.Hub.BroadcastToRoom("general", framework.H{
		"event": "message",
		"data":  string(msg),
	})
	return nil
}

func (ch *ChatChannel) OnDisconnect(ctx *websocket.WSContext) error {
	ctx.LeaveRoom("general")
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Background Jobs
// ──────────────────────────────────────────────────────────────────────────────

func handleWelcomeEmailJob(ctx context.Context, task *asynq.Task) error {
	fmt.Println("[Job] Sending welcome email...")
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Main — Wire everything together
// ──────────────────────────────────────────────────────────────────────────────

func main() {
	app := framework.New()

	// Register plugins
	app.Register(&healthcheck.Plugin{})

	// Initialize session store
	auth.InitSession(app.Config.App.SecretKeyBase)

	// Set up ORM-related items (these would work with a real database)
	_ = orm.Query[User]
	_ = orm.Query[Post]
	_ = orm.Query[Comment]

	// Set up cache (in-memory for demo)
	memCache := cache.NewMemoryAdapter()
	app.Cache = memCache

	// Set up i18n
	_ = i18n.T("welcome", i18n.Vars{"name": "World"})

	// Set up mailer
	welcomeMailer := WelcomeMailer{mailer.Mailer{Config: config.MailerConfig{
		SMTPHost: "localhost",
		SMTPPort: 1025,
		From:     "noreply@example.com",
	}}}

	// Set up WebSocket hub
	hub := websocket.NewHub()

	// Mount admin panel
	adminPanel := admin.Panel(admin.Config{
		Models: []admin.Resource{
			admin.NewResource[User]().WithSearchFields("Name", "Email"),
			admin.NewResource[Post]().WithSearchFields("Title"),
		},
		Auth: admin.BasicAuth("admin", "password"),
	})

	// Mount job dashboard
	jobDash := dashboard.Dashboard(dashboard.DashboardConfig{
		RedisAddr: "localhost:6379",
	})

	// Set up queue
	_ = queue.NewEnqueuer(config.RedisConfig{URL: "redis://localhost:6379"})

	// Routes
	app.Routes(func(r *framework.Router) {
		// RESTful resources
		r.Resources("users", &UsersController{})
		r.Resources("posts", &PostsController{}, func(r *framework.Router) {
			r.Resources("comments", &CommentsController{})
		})

		// Auth routes
		r.POST("/login", func(ctx *framework.Context) error {
			var input struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := ctx.Bind(&input); err != nil {
				return err
			}
			token, err := auth.GenerateToken(1)
			if err != nil {
				return ctx.InternalError(err)
			}
			return ctx.JSON(http.StatusOK, framework.H{"token": token})
		})

		// Protected routes
		r.Namespace("/api", func(r *framework.Router) {
			r.GET("/me", auth.Required(func(ctx *framework.Context) error {
				return ctx.JSON(http.StatusOK, framework.H{"user": ctx.CurrentUser()})
			}))
		})

		// WebSocket route
		r.WebSocket("/ws/chat", hub.HandleChannel(&ChatChannel{}))

		// Mailer demo
		r.POST("/demo/email", func(ctx *framework.Context) error {
			email := welcomeMailer.WelcomeEmail("Test User", "test@example.com")
			email.Deliver()
			return ctx.JSON(http.StatusOK, framework.H{"status": "email sent"})
		})

		// HTMX check
		r.GET("/demo/partial", func(ctx *framework.Context) error {
			if ctx.IsHTMX() {
				return ctx.JSON(http.StatusOK, framework.H{"partial": true})
			}
			return ctx.JSON(http.StatusOK, framework.H{"partial": false, "full_page": true})
		})

		// Mount plugin handlers
		r.Mount("/admin", adminPanel)
		r.Mount("/jobs", jobDash)
	})

	// Register job handlers (for worker mode)
	_ = handleWelcomeEmailJob

	app.Run()
}
