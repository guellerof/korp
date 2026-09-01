.PHONY: build test vet up down logs clean validate metrics load

build:
	cd app && go build ./...

test:
	cd app && go test ./...

vet:
	cd app && go vet ./...

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

clean:
	docker compose down --remove-orphans

validate:
	curl -fsS http://localhost/projeto-korp

metrics:
	curl -fsS http://localhost/metrics | grep -E 'http_requests_total|http_request_duration_seconds'

load:
	@i=0; while [ $$i -lt 100 ]; do curl -fsS http://localhost/projeto-korp >/dev/null; i=$$((i+1)); done
	@echo "100 requisicoes enviadas para /projeto-korp"
