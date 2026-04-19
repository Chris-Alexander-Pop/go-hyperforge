package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/rest"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/jwt"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	mfamemory "github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/social"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/webauthn"
	webauthnmemory "github.com/chris-alexander-pop/system-design-library/pkg/auth/webauthn/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/config"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/sql/adapters/sqlite"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

type SocialConfig struct {
	GoogleClientID     string `env:"SOCIAL_GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"SOCIAL_GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `env:"SOCIAL_GOOGLE_REDIRECT_URL" env-default:"http://localhost:8080/v1/auth/social/google/callback"`
}

type Config struct {
	Logger   logger.Config
	REST     rest.Config
	JWT      jwt.Config
	MFA      mfa.Config
	WebAuthn webauthn.Config
	Social   SocialConfig
	SQL      sql.Config
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	logr := logger.Init(cfg.Logger)
	ctx := context.Background()

	// Initialize SQL Database
	cfg.SQL.Driver = "sqlite" // ensure the driver is correct for the adapter
	dbAdapter, err := sqlite.New(cfg.SQL)
	if err != nil {
		log.Fatalf("failed to initialize sqlite: %v", err)
	}
	db := dbAdapter.Get(ctx)
	
	// AutoMigrate the user schema
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatalf("failed to migrate schema: %v", err)
	}

	repo := NewDBRepository(db)

	// Seed an initial user if they don't exist explicitly for local development
	if _, err := repo.GetUserByUsername(ctx, "admin"); err != nil {
		_, _ = repo.CreateUser(ctx, "admin", "admin@hyperforge.local", "password", "admin")
	}

	// Initialize Auth Adapters
	jwtAdapter := jwt.New(cfg.JWT)
	mfaAdapter := mfamemory.New(cfg.MFA)
	webAuthnAdapter := webauthnmemory.New(cfg.WebAuthn)

	googleSocial, err := social.New(social.ProviderGoogle, cfg.Social.GoogleClientID, cfg.Social.GoogleClientSecret, cfg.Social.GoogleRedirectURL)
	if err != nil {
		logr.WarnContext(ctx, "failed to initialize generic social oauth", "error", err)
	}
	socialAdapters := map[string]social.Provider{"google": googleSocial}

	// Initialize API Server
	server := rest.New(cfg.REST)

	bindDeps := HandlerDependencies{
		JWT:            jwtAdapter,
		MFA:            mfaAdapter,
		WebAuthn:       webAuthnAdapter,
		SocialAdapters: socialAdapters,
		Repo:           repo,
	}

	BindHandlers(server.Echo(), bindDeps)

	go func() {
		if err := server.Start(); err != nil {
			logr.ErrorContext(ctx, "server async err", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logr.InfoContext(ctx, "shutting down server gracefully")
	_ = server.Shutdown(ctx)
	_ = dbAdapter.Close()
}
