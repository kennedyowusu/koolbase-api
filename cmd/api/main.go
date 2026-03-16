package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

	"github.com/kennedyowusu/hatchway-api/internal/admin"
	"github.com/kennedyowusu/hatchway-api/internal/auditlog"
	"github.com/kennedyowusu/hatchway-api/internal/analytics"
	"github.com/kennedyowusu/hatchway-api/internal/auth"
	"github.com/kennedyowusu/hatchway-api/internal/bootstrap"
	"github.com/kennedyowusu/hatchway-api/internal/configs"
	"github.com/kennedyowusu/hatchway-api/internal/environments"
	"github.com/kennedyowusu/hatchway-api/internal/flags"
	"github.com/kennedyowusu/hatchway-api/internal/invitations"
	organizations "github.com/kennedyowusu/hatchway-api/internal/organization"
	projects "github.com/kennedyowusu/hatchway-api/internal/project"
	"github.com/kennedyowusu/hatchway-api/internal/versions"

	"github.com/kennedyowusu/hatchway-api/platform/cache"
	"github.com/kennedyowusu/hatchway-api/platform/db"
	"github.com/kennedyowusu/hatchway-api/platform/email"
	"github.com/kennedyowusu/hatchway-api/platform/events"
	apimiddleware "github.com/kennedyowusu/hatchway-api/platform/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("no .env file found, reading from environment")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	database, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer database.Close()

	rdb, err := cache.Connect(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer rdb.Close()

	bus := events.New()

	var mailer email.Provider
	if apiKey := os.Getenv("RESEND_API_KEY"); apiKey != "" {
		from := os.Getenv("EMAIL_FROM")
		if from == "" {
			from = "Koolbase <noreply@koolbase.com>"
		}
		mailer = email.NewResend(apiKey, from)
	} else {
		mailer = &email.NoopProvider{}
		log.Warn().Msg("email provider: noop (RESEND_API_KEY not set)")
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3001"
	}

	bootstrapHandler := bootstrap.NewHandler(database, rdb)
	adminHandler := admin.NewHandler(database, rdb)
	orgHandler := organizations.NewHandler(database)
	auditWriter := auditlog.NewWriter(database)
	projectHandler := projects.NewHandler(database)
	envHandler := environments.NewHandler(database, rdb)
	flagHandler := flags.NewHandler(database, bus)
	configHandler := configs.NewHandler(database, rdb)
	versionHandler := versions.NewHandler(database)

	authRepo := auth.NewPostgresRepository(database)
	authService := auth.NewService(authRepo, orgHandler, mailer, bus, appURL)
	authHandler := auth.NewHandler(authService)
	auth.StartCleanupJob(authRepo)
	inviteHandler := invitations.NewHandler(database, mailer, appURL)
	analyticsHandler := analytics.NewHandler(database)

	// 5 requests per minute, burst of 10
	authLimiter := apimiddleware.NewIPRateLimiter(rate.Every(time.Minute/5), 10)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(apimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins(),
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Internal-Key"},
		MaxAge:         300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// All /v1 routes in one block
	r.Route("/v1", func(r chi.Router) {
		// Public — no auth required
		r.Get("/bootstrap", bootstrapHandler.Handle)

		// Auth routes
		r.With(apimiddleware.RateLimit(authLimiter)).Post("/auth/signup", authHandler.Signup)
		r.With(apimiddleware.RateLimit(authLimiter)).Post("/auth/login", authHandler.Login)
		r.Post("/auth/logout", authHandler.Logout)
		r.Post("/auth/verify-email", authHandler.VerifyEmail)
		r.With(apimiddleware.RateLimit(authLimiter)).Post("/auth/forgot-password", authHandler.ForgotPassword)
		r.Post("/auth/reset-password", authHandler.ResetPassword)
			r.Post("/auth/verify-email-change", authHandler.ConfirmEmailChange)
			r.Post("/invites/peek", inviteHandler.PeekInvite)
			r.Post("/invites/accept", inviteHandler.ValidateInvite)

		// Management routes — protected by JWT
		r.Group(func(r chi.Router) {
			r.Use(apimiddleware.RequireAuth(authService))
					r.Use(apimiddleware.AuditLog(auditWriter))

			r.Post("/organizations", orgHandler.Create)
			r.Get("/organizations", orgHandler.List)
				r.Get("/organizations/{org_id}", orgHandler.Get)
				r.Patch("/organizations/{org_id}", orgHandler.Update)
				r.Get("/me", authHandler.Me)
				r.Patch("/me", authHandler.RequestEmailChange)
				r.Patch("/me/password", authHandler.ChangePassword)
					r.Delete("/me", authHandler.DeleteAccount)
					r.Get("/organizations/{org_id}/members", inviteHandler.ListMembers)
					r.Delete("/organizations/{org_id}/members/{user_id}", inviteHandler.RemoveMember)
					r.Post("/organizations/{org_id}/invites", inviteHandler.Invite)
					r.Get("/organizations/{org_id}/invites", inviteHandler.ListInvites)
					r.Get("/organizations/{org_id}/audit-logs", auditWriter.HandleList)
					r.Get("/organizations/{org_id}/analytics", analyticsHandler.GetOrgStats)
					r.Delete("/organizations/{org_id}/invites/{invite_id}", inviteHandler.RevokeInvite)
			r.Post("/organizations/{org_id}/projects", projectHandler.Create)
			r.Get("/organizations/{org_id}/projects", projectHandler.List)
			r.Post("/projects/{project_id}/environments", envHandler.Create)
			r.Get("/projects/{project_id}/environments", envHandler.List)
					r.Delete("/environments/{env_id}", envHandler.Delete)
			r.Post("/environments/{env_id}/flags", flagHandler.Create)
			r.Get("/environments/{env_id}/flags", flagHandler.List)
			r.Put("/flags/{flag_id}", flagHandler.Update)
			r.Delete("/flags/{flag_id}", flagHandler.Delete)
			r.Post("/environments/{env_id}/configs", configHandler.Create)
			r.Get("/environments/{env_id}/configs", configHandler.List)
			r.Put("/configs/{config_id}", configHandler.Update)
			r.Delete("/configs/{config_id}", configHandler.Delete)
			r.Put("/environments/{env_id}/version", versionHandler.Upsert)
			r.Get("/environments/{env_id}/version", versionHandler.List)
		})
	})

	// Internal service-to-service routes
	r.Route("/internal", func(r chi.Router) {
		r.Use(apimiddleware.InternalOnly)
		r.Post("/environments/{environment_id}/snapshot/rebuild", adminHandler.RebuildSnapshot)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("port", port).Msg("Koolbase API starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	<-quit
	log.Info().Msg("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}

	log.Info().Msg("server exited cleanly")
}

func allowedOrigins() []string {
	origins := []string{"http://localhost:3000", "http://localhost:3001"}
	if appURL := os.Getenv("APP_URL"); appURL != "" {
		origins = append(origins, appURL)
	}
	if landingURL := os.Getenv("LANDING_URL"); landingURL != "" {
		origins = append(origins, landingURL)
	}
	return origins
}
