package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

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
	})
	if err != nil {
		slog.Error("failed to create app", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", port)
	slog.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, application); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
