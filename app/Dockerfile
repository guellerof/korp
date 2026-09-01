FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/http-server-projeto-korp .

FROM scratch
COPY --from=builder /out/http-server-projeto-korp /http-server-projeto-korp
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD ["/http-server-projeto-korp", "-healthcheck"]
ENTRYPOINT ["/http-server-projeto-korp"]
