package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	"github.com/ramisoul84/assistant-server/internal/config"
	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/server/http/handler"
	"github.com/ramisoul84/assistant-server/internal/server/http/middleware"
	"github.com/ramisoul84/assistant-server/pkg/logger"
)

type Server struct {
	app *fiber.App
	cfg *config.Config
	log logger.Logger
}

func New(cfg *config.Config) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout: cfg.Server.ReadTimeout, WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout: cfg.Server.IdleTimeout, BodyLimit: cfg.Server.BodyLimitMB * 1024 * 1024,
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := 500
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})
	s := &Server{app: app, cfg: cfg, log: logger.Get()}
	s.app.Use(requestid.New())
	s.app.Use(func(c *fiber.Ctx) error {
		rid := c.GetRespHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.SetUserContext(context.WithValue(c.Context(), domain.RequestIDKey, rid))
		return c.Next()
	})
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Authorization",
		AllowCredentials: true, MaxAge: 86400,
	}))
	s.app.Use(compress.New())
	return s
}

func (s *Server) App() *fiber.App { return s.app }

func (s *Server) RegisterRoutes(auth *handler.AuthHandler, fin *handler.FinanceHandler, notes *handler.NoteHandler) {
	api := s.app.Group("/api/v1")
	api.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	// Public
	api.Post("/auth/request-otp", auth.RequestOTP)
	api.Post("/auth/verify-otp", auth.VerifyOTP)

	// Protected
	p := api.Group("", middleware.RequireAuth(s.cfg.Auth.JWTSecret))

	p.Get("/expenses", fin.ListExpenses)
	p.Post("/expenses", fin.CreateExpense)
	p.Put("/expenses/:id", fin.UpdateExpense)
	p.Delete("/expenses/:id", fin.DeleteExpense)

	p.Get("/incomes", fin.ListIncomes)
	p.Post("/incomes", fin.CreateIncome)
	p.Put("/incomes/:id", fin.UpdateIncome)
	p.Delete("/incomes/:id", fin.DeleteIncome)

	p.Get("/budget", fin.GetBudget)
	p.Put("/budget", fin.UpsertBudget)

	p.Get("/notes", notes.List)
	p.Post("/notes", notes.Create)
	p.Put("/notes/:id", notes.Update)
	p.Delete("/notes/:id", notes.Delete)
}
