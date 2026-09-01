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
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type projetoResponse struct {
	Nome    string `json:"nome"`
	Horario string `json:"horario"`
}

type metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total", Help: "Total de requisicoes HTTP da aplicacao."}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "Duracao das requisicoes HTTP da aplicacao em segundos.", Buckets: prometheus.DefBuckets}, []string{"method", "route", "status"}),
	}
	reg.MustRegister(m.requests, m.duration)
	return m
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func routeLabel(path string) string {
	switch path {
	case "/projeto-korp", "/healthz":
		return path
	default:
		return "unknown"
	}
}

func instrument(m *metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		route := routeLabel(r.URL.Path)
		status := strconv.Itoa(recorder.status)
		m.requests.WithLabelValues(r.Method, route, status).Inc()
		m.duration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())
	})
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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s remote=%s duration=%s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

func newServer() *http.Server {
	registry := prometheus.NewRegistry()
	m := newMetrics(registry)
	appMux := http.NewServeMux()
	appMux.HandleFunc("/projeto-korp", projetoHandler(time.Now))
	appMux.HandleFunc("/healthz", healthHandler)

	rootMux := http.NewServeMux()
	rootMux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	rootMux.Handle("/", instrument(m, appMux))
	return &http.Server{Addr: ":8080", Handler: logging(rootMux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
}

func runHealthcheck() error {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned %s", resp.Status)
	}
	return nil
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check local service health")
	flag.Parse()
	if *healthcheck {
		if err := runHealthcheck(); err != nil {
			log.Fatal(err)
		}
		return
	}

	srv := newServer()
	errCh := make(chan error, 1)
	go func() {
		log.Printf("http-server-projeto-korp listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
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
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		_ = srv.Close()
	}
}
