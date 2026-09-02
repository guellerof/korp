# http-server-projeto-korp

Desafio técnico Korp — serviço HTTP em Go, containerizado, publicado por NGINX, monitorado com Prometheus/Grafana e provisionado com Ansible.

## Arquitetura

```text
Cliente -> localhost:80 -> NGINX -> http-server-projeto-korp:8080
                                      |
                                      +-> /projeto-korp
                                      +-> /healthz
                                      +-> /metrics <- Prometheus:9090 <- Grafana:3000
```

Todos os containers compartilham a rede bridge explícita `korp-network`. A aplicação não publica a porta 8080 no host; somente o NGINX é o ponto de entrada da aplicação. Prometheus e Grafana publicam 9090 e 3000 para demonstração e análise do desafio.

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

Prometheus coleta `http-server-projeto-korp:8080/metrics` a cada 15 segundos. A métrica nativa `up{job="http-server-projeto-korp"}` representa a disponibilidade do target.

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
├── ansible/
│   ├── inventory.ini
│   └── playbook.yml
├── app/
│   ├── .dockerignore
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
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

## Provisionamento com Ansible

O caminho principal de execução do desafio é o playbook Ansible. Ele suporta explicitamente Ubuntu e Debian e deve ser executado por um usuário com acesso a `sudo`.

Na máquina Linux, instale apenas o Ansible e execute, a partir da raiz do repositório:

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml --ask-become-pass
```

Em ambientes nos quais o usuário já possui `sudo` sem senha, pode ser usado:

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml
```

Com um único comando, o playbook:

1. valida se o host é Ubuntu ou Debian;
2. instala e configura o repositório oficial do Docker;
3. instala Docker Engine, Buildx e Docker Compose v2;
4. habilita e inicia o Docker;
5. copia aplicação e configurações para `/opt/http-server-projeto-korp`;
6. cria a rede bridge `korp-network` somente quando ela ainda não existe;
7. valida o arquivo Compose;
8. reconcilia a stack com `docker compose up -d --build --remove-orphans`;
9. aguarda o serviço responder através do NGINX;
10. exibe no console a resposta JSON de `/projeto-korp`;
11. valida readiness do Prometheus e health do Grafana.

A rede é criada explicitamente pelo Ansible e declarada como `external` no Compose. Isso deixa clara a separação de responsabilidades: Ansible provisiona a infraestrutura Docker e o Compose gerencia os containers da aplicação.

O playbook pode ser executado novamente com segurança: pacotes, arquivos, serviço Docker e rede são tratados de forma declarativa ou condicional, enquanto o `docker compose up` reconcilia o estado desejado dos containers. Como a CLI do Compose não fornece ao Ansible uma indicação estável de mudança em todas as versões, essa etapa é reportada como `ok`; o estado real é validado pelos endpoints ao final do playbook.

Durante a validação final, o playbook foi executado duas vezes consecutivas no mesmo ambiente. A segunda execução concluiu com `changed=0`, `unreachable=0` e `failed=0`, confirmando a reexecução segura do provisionamento.

## Execução manual com Docker

Como a rede é provisionada explicitamente, para executar o Compose sem Ansible crie-a primeiro:

```bash
docker network inspect korp-network >/dev/null 2>&1 || docker network create --driver bridge korp-network
docker compose up -d --build
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

Para verificar a stack provisionada pelo Ansible e confirmar que a aplicação continua sem publicar 8080 no host:

```bash
docker compose -f /opt/http-server-projeto-korp/docker-compose.yml ps
```

## Testes

```bash
make test
make vet
```

## Decisões técnicas

- **Ansible como orquestrador do provisionamento:** prepara o host, instala Docker, cria a rede e executa/valida a stack.
- **Reexecução segura:** módulos declarativos são usados para pacotes, arquivos e serviços; a rede só é criada quando não existe e o Compose reconcilia os containers.
- **Rede bridge explícita:** `korp-network` é criada pelo Ansible e consumida pelo Compose como rede externa.
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
