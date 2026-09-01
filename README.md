# http-server-projeto-korp

Desafio técnico Korp — serviço HTTP em Go executado em containers Docker e publicado por NGINX como reverse proxy.

## Arquitetura

```text
Cliente -> localhost:80 -> NGINX -> http-server-projeto-korp:8080
                                      |
                                      +-> /projeto-korp
                                      +-> /healthz
```

Os dois containers compartilham a rede Docker `korp-network`, do tipo `bridge`. A aplicação não publica a porta 8080 no host; apenas o NGINX publica a porta 80.

## API

### GET /projeto-korp

Retorna JSON com o nome do projeto e o horário UTC calculado a cada requisição:

```json
{
  "nome": "Projeto Korp",
  "horario": "2026-09-01T22:00:00Z"
}
```

### GET /healthz

Endpoint de disponibilidade utilizado pelos healthchecks.

## Estrutura

```text
.
├── app/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   └── main_test.go
├── nginx/
│   └── http-server-projeto-korp.conf
├── .dockerignore
├── .gitignore
├── docker-compose.yml
├── Makefile
└── README.md
```

## Pré-requisitos

- Linux
- Docker Engine
- Docker Compose v2
- Go 1.24+ apenas para desenvolvimento/testes locais; não é necessário para executar via Docker

## Execução

```bash
docker compose up -d --build
```

ou:

```bash
make up
```

## Validação

```bash
curl http://localhost:80/projeto-korp
```

A aplicação deve responder com HTTP 200 e JSON contendo `nome` e `horario`.

Verifique que a aplicação não publica a porta 8080 no host:

```bash
docker compose ps
```

O acesso externo ao serviço ocorre exclusivamente pelo NGINX.

## Testes

```bash
make test
make vet
```

## Decisões técnicas

- **Go standard library:** suficiente para o serviço simples e reduz dependências externas.
- **Horário injetável no handler:** mantém o horário dinâmico em produção e permite testes determinísticos.
- **Graceful shutdown:** trata SIGINT/SIGTERM e concede até 10 segundos para conexões em andamento.
- **Timeouts HTTP:** evita conexões indefinidamente abertas.
- **Logging:** registra método, caminho, endereço remoto e duração.
- **Multi-stage build:** compila em uma imagem Go e executa somente o binário na imagem final `scratch`.
- **Non-root:** o processo roda como UID/GID `65532:65532`.
- **Healthcheck sem ferramentas extras:** o próprio binário implementa `-healthcheck`, preservando a imagem `scratch`.
- **Rede bridge explícita:** comunicação entre containers ocorre pela `korp-network` usando DNS interno do Compose.
- **NGINX como único ponto de entrada:** a porta 8080 usa apenas `expose`; somente a porta 80 do NGINX é publicada no host.
- **Configuração NGINX por diretório:** `./nginx` é montado em `/etc/nginx/conf.d:ro`, ocultando a configuração default da imagem.

## Comandos úteis

```bash
make build
make test
make vet
make up
make logs
make validate
make down
```

## Próximas etapas do desafio

As próximas partes adicionarão Prometheus/Grafana para observabilidade e Ansible para provisionamento automatizado do ambiente.
