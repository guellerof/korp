package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProjetoHandler(t *testing.T) {
	fixed := time.Date(2026, 9, 1, 19, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	req := httptest.NewRequest(http.MethodGet, "/projeto-korp", nil)
	rr := httptest.NewRecorder()
	projetoHandler(func() time.Time { return fixed })(rr, req)
	if rr.Code != http.StatusOK { t.Fatalf("status: got %d", rr.Code) }
	if got := rr.Header().Get("Content-Type"); got != "application/json" { t.Fatalf("content-type: got %q", got) }
	var body projetoResponse
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil { t.Fatal(err) }
	if body.Nome != "Projeto Korp" { t.Fatalf("nome: got %q", body.Nome) }
	if body.Horario != "2026-09-01T22:00:00Z" { t.Fatalf("horario: got %q", body.Horario) }
}

func TestProjetoHandlerRejectsNonGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/projeto-korp", nil)
	rr := httptest.NewRecorder()
	projetoHandler(time.Now)(rr, req)
	if rr.Code != http.StatusMethodNotAllowed { t.Fatalf("status: got %d", rr.Code) }
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	healthHandler(rr, req)
	if rr.Code != http.StatusOK { t.Fatalf("status: got %d", rr.Code) }
}

func TestServerTimeouts(t *testing.T) {
	srv := newServer()
	if srv.ReadHeaderTimeout <= 0 || srv.ReadTimeout <= 0 || srv.WriteTimeout <= 0 || srv.IdleTimeout <= 0 { t.Fatal("expected all HTTP timeouts to be configured") }
}
