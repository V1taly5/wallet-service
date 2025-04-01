package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wallet-service/internal/api"
	"wallet-service/internal/config"
	"wallet-service/internal/repository"
	"wallet-service/internal/service"

	"github.com/kelseyhightower/envconfig"

	_ "github.com/lib/pq"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	config.LoadEnv()
	var config config.Config
	err := envconfig.Process("", &config)
	if err != nil {
		return
	}
	fmt.Println(config)
	db, err := initDatabase(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)

	}
	defer db.Close()

	logger := setupLogger(config.Env)

	walletRepo := repository.NewWalletRepository(db, logger)

	if err = walletRepo.CreateTabeIfNotExists(context.Background()); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	walletService := service.NewWalletService(walletRepo, logger)

	router := api.NewRouter(walletService)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ServerPort),
		Handler: router,
	}

	go func() {
		log.Printf("Starting server on port %d", config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func initDatabase(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DataBase.URL)
	fmt.Println(cfg.DataBase.URL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.ConnectionPool.MaxOpenConns)
	db.SetMaxIdleConns(cfg.ConnectionPool.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnectionPool.MaxLifetime) * time.Second)

	return db, nil
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return log
}
