package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alnvdl/autosave"

	"github.com/alnvdl/anything/internal/app"
)

// serverHealthCheck periodically checks the server health by making a request
// to the /status endpoint.
func serverHealthCheck(interval time.Duration, port int, close chan bool) {
	for {
		select {
		case <-time.After(interval):
			res, err := http.Get(fmt.Sprintf("http://localhost:%d/status", port))
			if err != nil {
				slog.Error("error making health check request",
					slog.String("err", err.Error()))
				continue
			}
			if res.StatusCode == http.StatusOK {
				slog.Info("server is healthy")
			} else {
				slog.Error("server is not healthy",
					slog.Int("status_code", res.StatusCode))
			}
		case <-close:
			slog.Info("stopping health check mechanism")
			return
		}
	}
}

func main() {
	port, err := Port()
	if err != nil {
		slog.Error("failed to read PORT", "error", err)
		os.Exit(1)
	}

	entries, err := Entries()
	if err != nil {
		slog.Error("failed to read ENTRIES", "error", err)
		os.Exit(1)
	}

	people, err := People()
	if err != nil {
		slog.Error("failed to read PEOPLE", "error", err)
		os.Exit(1)
	}

	tz, err := Timezone()
	if err != nil {
		slog.Error("failed to read TIMEZONE", "error", err)
		os.Exit(1)
	}

	periods, err := Periods()
	if err != nil {
		slog.Error("failed to read PERIODS", "error", err)
		os.Exit(1)
	}

	application, err := app.New(app.Params{
		Entries:  entries,
		People:   people,
		Timezone: tz,
		Periods:  periods,
		AutoSaveParams: autosave.Params{
			FilePath: DBPath(),
			Interval: PersistInterval(),
			Logger:   slog.Default(),
		},
	})
	if err != nil {
		slog.Error("failed to create app", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: application,
	}

	healthCheck := make(chan bool)
	go serverHealthCheck(HealthCheckInterval(), port, healthCheck)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		close(healthCheck)
		application.Close()
		slog.Info("shutting down server")
		server.Shutdown(context.Background())
	}()

	slog.Info("starting server", "addr", addr)
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			slog.Info("server shut down")
		} else {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}
}
