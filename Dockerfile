# Etapa de build
FROM docker.io/golang:1.24.6-bookworm AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
# Genera binario estático (útil para runtime mínimo)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/hello-go ./...

# Runtime no-root compatible con OpenShift
FROM gcr.io/distroless/static:nonroot
USER 65532:65532
WORKDIR /app
COPY --from=builder /app/hello-go /app/hello-go
EXPOSE 8080
ENTRYPOINT ["/app/hello-go"]
