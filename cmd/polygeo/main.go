package main

import (
	"log/slog"
	"os"

	"github.com/kerbymart/polygeo/internal/geo"
	"github.com/kerbymart/polygeo/internal/httpapi"
)

func main() {
	dataDirectory := envOrDefault("POLYGEO_DATA_DIR", "./data")
	address := envOrDefault("POLYGEO_ADDR", ":8080")

	registry, err := geo.LoadDirectory(dataDirectory)
	if err != nil {
		slog.Error("failed to load country data", "directory", dataDirectory, "error", err)
		os.Exit(1)
	}

	server := httpapi.New(registry)
	slog.Info("starting Polygeo", "address", address, "data_directory", dataDirectory, "countries", registry.Count())
	if err := server.Start(address); err != nil {
		slog.Error("Polygeo stopped", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
