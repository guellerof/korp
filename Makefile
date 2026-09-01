.PHONY: build test vet up down logs clean validate

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
