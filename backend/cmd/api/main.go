// Command api is the BDIC school management API.
//
// It reads .env, connects to PostgreSQL, applies any pending migrations,
// creates the first administrator if the database is empty, and serves HTTP
// until it is asked to stop.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bdic/backend/internal/api"
	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/config"
	"bdic/backend/internal/db"
	"bdic/backend/internal/files"
	"bdic/backend/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("the API could not start", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// .env is read before the config, and never overrides a variable that is
	// already set, so a value exported in the shell or supplied by the
	// deployment always wins over the file.
	// `make api` runs this command from backend/, while setup creates the
	// repository-wide .env at the project root. Production deployments supply
	// variables directly, so this local file is only a development convenience.
	if err := config.LoadEnvFile("../.env"); err != nil {
		slog.Warn("could not read .env", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	logger.Info("starting BDIC API",
		"env", cfg.Env,
		"port", cfg.Port,
		"corsOrigins", cfg.CORSOrigins)

	// A generous boot timeout: the first start applies ten migrations, and on a
	// laptop that is slower than a warm restart.
	bootCtx, cancelBoot := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelBoot()

	pool, err := db.Open(bootCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(bootCtx, pool, logger); err != nil {
		return err
	}

	dataStore := store.New(pool)
	recorder := audit.NewRecorder(pool, logger)
	signer := auth.NewSigner(cfg.SigningKey, cfg.TokenTTL)

	// Uploaded documents and photographs. Local disk here; S3 once deployed,
	// which is a different Storage implementation and no handler changes.
	storage, err := files.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		return err
	}
	logger.Info("file storage ready", "directory", storage.Root())

	if err := ensureBootstrapAdmin(bootCtx, cfg, dataStore, logger); err != nil {
		return err
	}

	server := api.New(cfg, dataStore, signer, recorder, storage, logger)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.Handler(),
		// Timeouts, because a connection that never finishes sending its
		// request would otherwise hold a goroutine open forever.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Shut down on Ctrl-C or on the signal a container runtime sends, so an
	// in-flight fee receipt is finished rather than cut off.
	shutdownCtx, stopListening := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopListening()

	// Buffered and never closed: closing it would let the select below receive
	// a nil error from a dead listener and return as if all were well.
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("listening", "url", "http://localhost:"+cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-shutdownCtx.Done():
		logger.Info("shutting down")
		graceCtx, cancelGrace := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancelGrace()
		if err := httpServer.Shutdown(graceCtx); err != nil {
			logger.Error("forced shutdown", "error", err)
			return err
		}
		logger.Info("stopped cleanly")
	}

	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	level := slog.LevelDebug
	if cfg.IsProduction() {
		level = slog.LevelInfo
	}

	options := &slog.HandlerOptions{Level: level}

	// JSON in production because a log aggregator reads it; text locally
	// because a person does.
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}

// ensureBootstrapAdmin creates the very first login so somebody can sign in and
// create everyone else.
//
// It runs only when the users table is completely empty. Once a real
// administrator exists this does nothing, and the values should be removed
// from .env. In production the config refuses to start if they are still set.
func ensureBootstrapAdmin(
	ctx context.Context,
	cfg config.Config,
	dataStore *store.Store,
	logger *slog.Logger,
) error {
	count, err := dataStore.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if cfg.BootstrapAdmin.Email == "" || cfg.BootstrapAdmin.Password == "" {
		logger.Warn("no users exist and no bootstrap administrator is configured",
			"fix", "set BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD in .env, then restart")
		return nil
	}

	hashed, err := auth.HashPassword(cfg.BootstrapAdmin.Password)
	if err != nil {
		return err
	}

	userID, err := dataStore.CreateUser(ctx, store.NewUser{
		Email:        cfg.BootstrapAdmin.Email,
		PasswordHash: hashed,
		FullNameEN:   cfg.BootstrapAdmin.Name,
		Role:         auth.RoleSuperAdmin,
		Locale:       "en",
		// The bootstrap password came from a file that may well be in a chat
		// log or a terminal history, so it has to be changed on first use.
		MustReset: true,
	})
	if err != nil {
		return err
	}

	logger.Info("created the first administrator",
		"email", cfg.BootstrapAdmin.Email,
		"userId", userID,
		"next", "sign in, change this password, then remove BOOTSTRAP_ADMIN_* from .env")
	return nil
}
