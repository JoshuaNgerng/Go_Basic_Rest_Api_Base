package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"authapi/internal/auth"
	"authapi/internal/config"
	"authapi/internal/db"
	"authapi/internal/server"
	"authapi/internal/user"
)

// main is the composition root: the only place that knows about every package.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	queries := db.New(pool)

	authSvc, err := auth.NewService(queries, auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL), cfg.PBKDF2Iterations)
	if err != nil {
		log.Fatalf("auth service: %v", err)
	}
	userSvc := user.NewService(queries)

	// hourly cleanup of expired revoked tokens
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := authSvc.PurgeExpiredTokens(ctx); err != nil {
					log.Printf("cleanup: %v", err)
				}
			}
		}
	}()

	if err := server.New(cfg.Port, authSvc, userSvc).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
