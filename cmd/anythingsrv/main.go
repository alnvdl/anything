package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alnvdl/autosave"

	"github.com/alnvdl/anything/internal/app"
)

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

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
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
