# ------------------------------------------------------------------------------
# STAGE 1: The Builder
# ------------------------------------------------------------------------------
# We use the official Go image to compile the application.
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o crawler ./cmd/crawler

FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont

COPY --from=builder /app/crawler .

RUN addgroup -S crawlergroup && adduser -S crawleruser -G crawlergroup
USER crawleruser

CMD ["./crawler"]