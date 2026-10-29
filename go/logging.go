package main

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// Assigned once during startup, before any handlers or background jobs run.
// Services capture their own logger so tests do not mutate global logging state.
var appLogger = newLogger(os.Stderr, slog.LevelInfo)

func newLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}

func logIdentity(value string) string {
	const limit = 128
	if len(value) > limit {
		return value[:limit] + "…"
	}
	return value
}

func endpointIssueCount(rows []map[string]any) int {
	count := 0
	for _, row := range rows {
		switch issues := row["issues"].(type) {
		case []any:
			count += len(issues)
		case []string:
			count += len(issues)
		}
	}
	return count
}

func configuredLogger(w io.Writer, level, debug string) (*slog.Logger, error) {
	var threshold slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "":
		threshold = slog.LevelInfo
		if debug == "true" {
			threshold = slog.LevelDebug
		}
	case "debug":
		threshold = slog.LevelDebug
	case "info":
		threshold = slog.LevelInfo
	case "warn":
		threshold = slog.LevelWarn
	case "error":
		threshold = slog.LevelError
	default:
		return newLogger(w, slog.LevelError), errors.New("LOG_LEVEL must be debug, info, warn or error")
	}
	return newLogger(w, threshold), nil
}

// Marshal before committing the response so encoding failures can return 500.
// Error values can contain private data; only application-owned reasons are logged.
func writeJSON(w http.ResponseWriter, status int, value any, logger *slog.Logger, component string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	data, err := json.Marshal(value)
	if err != nil {
		logger.Error("Cannot encode response", "event", "http.encoding_failed", "component", component, "reason", "invalid response data")
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	if _, err := w.Write(append(data, '\n')); err != nil {
		logger.Debug("Cannot write response", "event", "http.write_failed", "component", component, "reason", "response writer unavailable")
	}
}
