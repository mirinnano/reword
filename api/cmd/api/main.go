package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mirinnano/reword/api/internal/app"
	"github.com/mirinnano/reword/api/internal/platform/config"
	"github.com/mirinnano/reword/api/internal/platform/db"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	a := app.New(app.Options{
		DB:           pool,
		Log:          log,
		AppOrigin:    cfg.AppOrigin,
		CookieSecure: cfg.CookieSecure,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           a.Handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// The job worker runs in-process; one API instance is enough for now and
	// SKIP LOCKED keeps it correct if more are added.
	workerCtx, stopWorker := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { a.Worker.Run(workerCtx) })

	errCh := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		stopWorker()
		wg.Wait()
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	stopWorker()
	wg.Wait()
	return err
}
