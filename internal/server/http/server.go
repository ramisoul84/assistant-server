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
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		IdleTimeout:           cfg.Server.IdleTimeout,
		BodyLimit:             cfg.Server.BodyLimitMB * 1024 * 1024,
		DisableStartupMessage: true,
		ProxyHeader:           "X-Forwarded-For",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})
	s := &Server{app: app, cfg: cfg, log: logger.Get()}
	s.registerMiddleware()
	return s
}

func (s *Server) App() *fiber.App { return s.app }

func (s *Server) registerMiddleware() {
	s.app.Use(requestid.New(requestid.Config{Header: "X-Request-ID"}))
	s.app.Use(func(c *fiber.Ctx) error {
		rid := c.GetRespHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.SetUserContext(context.WithValue(c.Context(), domain.RequestIDKey, rid))
		return c.Next()
	})
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     s.cfg.Server.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Authorization,X-Request-ID",
		AllowCredentials: true,
		MaxAge:           86400,
	}))
	s.app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
}

func (s *Server) RegisterRoutes(
	authH *handler.AuthHandler,
	apptH *handler.AppointmentHandler,
	expH *handler.ExpenseHandler,
	incomeH *handler.IncomeHandler,
	gymH *handler.GymHandler,
	budgetH *handler.BudgetHandler,
) {
	api := s.app.Group("/api/v1")
	api.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	// Public
	auth := api.Group("/auth")
	auth.Post("/request-otp", authH.RequestOTP)
	auth.Post("/verify-otp", authH.VerifyOTP)

	// Protected
	p := api.Group("", middleware.RequireAuth(s.cfg.Auth.JWTSecret))
	p.Get("/appointments", apptH.List)
	p.Post("/appointments", apptH.Create)
	p.Put("/appointments/:id", apptH.Update)
	p.Delete("/appointments/:id", apptH.Delete)

	p.Get("/expenses", expH.List)
	p.Post("/expenses", expH.Create)
	p.Put("/expenses/:id", expH.Update)
	p.Delete("/expenses/:id", expH.Delete)

	p.Get("/incomes", incomeH.List)
	p.Post("/incomes", incomeH.Create)
	p.Put("/incomes/:id", incomeH.Update)
	p.Delete("/incomes/:id", incomeH.Delete)

	p.Get("/gym/sessions", gymH.ListSessions)
	p.Delete("/gym/sessions/:id", gymH.DeleteSession)
	p.Put("/gym/exercises/:id", gymH.UpdateExercise)
	p.Delete("/gym/exercises/:id", gymH.DeleteExercise)

	p.Get("/budget", budgetH.Get)
	p.Put("/budget", budgetH.Upsert)
}
