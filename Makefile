.PHONY: build test vet network up down logs clean validate metrics load provision

build:
	cd app && go build ./...

test:
	cd app && go test ./...

vet:
	cd app && go vet ./...

network:
	@docker network inspect korp-network >/dev/null 2>&1 || docker network create --driver bridge korp-network

up: network
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

provision:
	ansible-playbook -i ansible/inventory.ini ansible/playbook.yml
