# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO for sqlite
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o jotaku-server ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/jotaku-server /app/jotaku-server

# Create data directory
RUN mkdir -p /data

VOLUME ["/data"]

EXPOSE 5689

ENV DB_PATH=/data/jotaku.db
ENV PORT=5689

CMD ["/app/jotaku-server"]
