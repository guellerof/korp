# http-server-projeto-korp

Desafio técnico Korp — serviço HTTP em Go, containerizado, publicado por NGINX e monitorado com Prometheus e Grafana.

## Arquitetura

```text
Cliente -> localhost:80 -> NGINX -> http-server-projeto-korp:8080
                                      |        ^
                                      |        |
                                      |     Prometheus:9090
                                      |        ^
                                      |        |
                                      |     Grafana:3000
                                      |
                                      +-> /projeto-korp
                                      +-> /healthz
                                      +-> /metrics
```

Todos os containers compartilham a rede bridge `korp-network`. A aplicação não publica a porta 8080 no host; somente o NGINX é o ponto de entrada da aplicação. Prometheus e Grafana publicam 9090 e 3000 para demonstração e análise do desafio.

## API

### GET /projeto-korp

Retorna JSON com o nome do projeto e o horário UTC calculado a cada requisição.

```json
{
  "nome": "Projeto Korp",
  "horario": "2026-09-01T22:00:00Z"
}
```

### GET /healthz

Endpoint dedicado de disponibilidade utilizado pelos healthchecks.

### GET /metrics

Expõe métricas no formato Prometheus. O scrape de `/metrics` não é contabilizado nas métricas de tráfego da aplicação.

Métricas customizadas:

- `http_requests_total{method,route,status}` — volume de requisições.
- `http_request_duration_seconds{method,route,status}` — histograma de latência.

As rotas são normalizadas (`/projeto-korp`, `/healthz` e `unknown`) para evitar labels de alta cardinalidade.

## Monitoramento

Prometheus coleta `http-server-projeto-korp:8080/metrics` a cada 15 segundos. A métrica nativa do Prometheus `up{job="http-server-projeto-korp"}` representa a disponibilidade do target.

Grafana recebe o datasource Prometheus e o dashboard automaticamente por provisioning. Não é necessário criar datasource ou importar JSON manualmente.

Dashboard `HTTP Server Projeto Korp`:

- disponibilidade (`up`);
- total de requisições;
- requisições por segundo;
- latência p95;
- taxa de requisições por status HTTP.

Principais queries:

```promql
up{job="http-server-projeto-korp"}
sum(http_requests_total{route="/projeto-korp"})
sum(rate(http_requests_total{route="/projeto-korp"}[5m]))
histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket{route="/projeto-korp"}[5m])))
sum by (status) (rate(http_requests_total{route="/projeto-korp"}[5m]))
```

## Estrutura

```text
.
├── app/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   └── main_test.go
├── grafana/
│   ├── dashboards/
│   │   └── http-server-projeto-korp-dashboard.json
│   └── provisioning/
│       ├── dashboards/dashboards.yml
│       └── datasources/datasource.yml
├── nginx/
│   └── http-server-projeto-korp.conf
├── prometheus/
│   └── prometheus.yml
├── docker-compose.yml
├── Makefile
└── README.md
```

## Pré-requisitos

- Linux
- Docker Engine
- Docker Compose v2
- Go 1.24+ somente para desenvolvimento/testes locais

## Execução

```bash
docker compose up -d --build
```

ou:

```bash
make up
```

## URLs

- Aplicação: `http://localhost/projeto-korp`
- Métricas via NGINX: `http://localhost/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

Para a demonstração local, o Grafana usa `admin/admin` por padrão. É possível substituir com `GRAFANA_ADMIN_USER` e `GRAFANA_ADMIN_PASSWORD`. Credenciais padrão não devem ser usadas em produção.

## Validação

```bash
curl -fsS http://localhost/projeto-korp
make metrics
make load
```

`make load` gera 100 requisições para `/projeto-korp`, facilitando a visualização dos gráficos.

No Prometheus, abra `Status > Target health` e confirme que o job `http-server-projeto-korp` está `UP`.

No Grafana, acesse a pasta `Korp` e abra o dashboard `HTTP Server Projeto Korp`.

Para verificar que a aplicação continua sem publicar 8080 no host:

```bash
docker compose ps
```

## Testes

```bash
make test
make vet
```

## Decisões técnicas

- **Go + biblioteca padrão:** mantém o serviço simples; `client_golang` é utilizado apenas para instrumentação Prometheus.
- **Métricas com baixa cardinalidade:** labels usam método, rota normalizada e status, evitando paths arbitrários.
- **Scrape fora da instrumentação:** `/metrics` não incrementa o contador de tráfego da própria aplicação.
- **Disponibilidade em duas camadas:** `/healthz` atende healthchecks e `up` mostra a capacidade real do Prometheus de coletar o serviço.
- **Dashboard como código:** datasource, provider e dashboard são provisionados automaticamente e versionados no Git.
- **Persistência:** Prometheus e Grafana utilizam volumes nomeados.
- **Multi-stage + scratch + non-root:** reduz a superfície da imagem final do serviço.
- **NGINX como único ponto de entrada da aplicação:** a porta 8080 permanece apenas na rede Docker.

## Comandos úteis

```bash
make build
make test
make vet
make up
make logs
make validate
make metrics
make load
make down
```

## Próxima etapa

A Parte 3 adicionará Ansible para instalar/configurar Docker, provisionar o ambiente completo e validar o serviço com um único comando.
