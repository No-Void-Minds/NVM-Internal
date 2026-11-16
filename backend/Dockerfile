# syntax=docker/dockerfile:1

# Samakan versi minor dengan baris `go` di go.mod.
FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Distroless static tidak punya libc. Binary hasil CGO gagal jalan dengan pesan "no such file or directory"
# walaupun berkasnya ada (kejadian di compro).
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed && \
    go build -trimpath -ldflags="-s -w" -o /out/import-excel ./cmd/import-excel

# Menggagalkan build saat itu juga, bukan menunggu container restart terus di VM (pola doomsday).
RUN for bin in /out/server /out/seed /out/import-excel; do \
      go version -m "$bin" | grep -q 'CGO_ENABLED=0' \
        || { echo "$bin tidak dibuild dengan CGO_ENABLED=0" >&2; exit 1; }; \
    done

# nonroot = UID dan GID 65532. Folder storage di host harus dimiliki UID ini (lihat docs/runbook-deploy.md).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /out/seed /out/import-excel /usr/local/bin/
USER nonroot:nonroot
ENV APP_ENV=production
ENV HTTP_ADDR=:8080
ENV STORAGE_DIR=/data/storage
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
