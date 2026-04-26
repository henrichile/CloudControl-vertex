package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/etasoft/cloudcontrol/internal/aiops"
	"github.com/etasoft/cloudcontrol/internal/api"
	"github.com/etasoft/cloudcontrol/internal/api/handlers"
	"github.com/etasoft/cloudcontrol/internal/config"
	"github.com/etasoft/cloudcontrol/internal/container"
	"github.com/etasoft/cloudcontrol/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	log.Info().Str("path", cfg.DatabasePath).Msg("database connected")

	mgr, err := container.NewManager(cfg.DockerHost)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Docker Engine")
	}
	defer mgr.Close()

	if _, err := mgr.List(context.Background(), false); err != nil {
		log.Warn().Err(err).Msg("Docker Engine ping failed — check DOCKER_HOST and permissions")
	} else {
		log.Info().Str("host", cfg.DockerHost).Msg("Docker Engine connected")
	}

	ollamaClient := aiops.NewClient(cfg.OllamaHost, cfg.OllamaModel)
	if err := ollamaClient.Ping(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Ollama not available — AIOps features will fail until Ollama is started")
	} else {
		log.Info().Str("model", cfg.OllamaModel).Msg("Ollama connected")
	}

	analyzer := aiops.NewAnalyzer(ollamaClient)
	engine := container.NewTemplateEngine()

	healthHandler := handlers.NewHealthHandler(
		func(ctx context.Context) error {
			_, err := mgr.List(ctx, false)
			return err
		},
		func(ctx context.Context) error {
			return ollamaClient.Ping(ctx)
		},
	)

	router := api.NewRouter(
		healthHandler,
		handlers.NewAuthHandler(db, cfg.JWTSecret),
		handlers.NewContainerHandler(mgr),
		handlers.NewProjectHandler(db, engine),
		handlers.NewAIOpsHandler(analyzer, mgr, db),
		cfg.JWTSecret,
	)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	ginEngine := gin.New()
	ginEngine.Use(gin.Recovery())
	ginEngine.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Msg("request")
	})

	router.Register(ginEngine)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      ginEngine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("addr", srv.Addr).Msg("Cloud Control API started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server stopped")
}
