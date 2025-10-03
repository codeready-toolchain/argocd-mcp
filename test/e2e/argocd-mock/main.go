package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/codeready-toolchain/argocd-mcp/test/resources"
)

func main() {
	listen := os.Getenv("ARGOCD_SERVER_LISTEN")
	token := os.Getenv("ARGOCD_SERVER_TOKEN")
	debug := os.Getenv("ARGOCD_SERVER_DEBUG")

	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	if debug == "true" {
		lvl.Set(slog.LevelDebug)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", token):
			logger.Debug("unauthorized request")
			w.WriteHeader(http.StatusUnauthorized)
			return
		case r.URL.Query().Get("name") == "example":
			logger.Debug("serving example application")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(resources.ExampleApplicationStr))
			return
		case r.URL.Query().Get("name") == "example-error":
			logger.Debug("serving example error application")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Debug("serving mock applications")
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resources.ApplicationsStr))
	})

	srv := &http.Server{
		Addr:         listen,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	logger.Info("serving argocd mock", "url", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
