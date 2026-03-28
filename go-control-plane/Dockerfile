FROM golang:1.26-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/go-control-plane ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libc6 \
    libgcc-s1 \
    libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/go-control-plane /usr/local/bin/go-control-plane

WORKDIR /app
EXPOSE 8080

ENTRYPOINT ["go-control-plane", "server"]
