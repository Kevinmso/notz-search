# syntax=docker/dockerfile:1

# ---- Build ----
# Mantenha a versão alinhada com a diretiva `toolchain` do go.mod.
FROM golang:1.27.1-alpine AS build
WORKDIR /src

# Dependências primeiro: essa camada só é refeita quando go.mod/go.sum mudam.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Binário estático (sem CGO), sem símbolos de debug.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/notz-search ./cmd

# ---- Runtime ----
# distroless/static: sem shell nem gerenciador de pacotes, mas com certificados
# CA (necessários para TLS com Qdrant Cloud e APIs de embeddings).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/notz-search /notz-search

# Usuário sem privilégios (nonroot = uid 65532).
USER 65532:65532

# O Render injeta PORT; 8080 é o fallback do código.
EXPOSE 8080
ENTRYPOINT ["/notz-search"]
