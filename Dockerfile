FROM golang:1.22-alpine3.20 AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev sqlite-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=1 go build -ldflags="-s -w -X github.com/bdfrost/poterie/internal/config.Version=${VERSION}" -o /poterie .

# Runtime
FROM alpine:3.20
RUN apk add --no-cache sqlite-libs

# Create data directory for SQLite
RUN mkdir -p /data && chown -R 1000:1000 /data

COPY --from=builder /poterie /usr/local/bin/poterie

EXPOSE 8080
USER 1000
ENTRYPOINT ["/usr/local/bin/poterie"]
