package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type projetoResponse struct {
	Nome    string `json:"nome"`
	Horario string `json:"horario"`
}

func projetoHandler(now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(projetoResponse{Nome: "Projeto Korp", Horario: now().UTC().Format(time.RFC3339)})
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s remote=%s duration=%s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

func newServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/projeto-korp", projetoHandler(time.Now))
	mux.HandleFunc("/healthz", healthHandler)
	return &http.Server{Addr: ":8080", Handler: logging(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
}

func runHealthcheck() error {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/healthz")
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("healthcheck returned %s", resp.Status) }
	return nil
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check local service health")
	flag.Parse()
	if *healthcheck {
		if err := runHealthcheck(); err != nil { log.Fatal(err) }
		return
	}

	srv := newServer()
	errCh := make(chan error, 1)
	go func() {
		log.Printf("http-server-projeto-korp listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { errCh <- err }
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		log.Printf("received signal %s; shutting down", sig)
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil { log.Printf("graceful shutdown failed: %v", err); _ = srv.Close() }
}
