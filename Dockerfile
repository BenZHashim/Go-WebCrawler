# Start with a small, lightweight Go image
# Start with a small, lightweight Go image
FROM golang:1.24-alpine

# Set the working directory inside the container
WORKDIR /app

# --- NEW: Install Chromium and dependencies ---
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont
# ---------------------------------------------

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy ALL your new files
COPY . .

# Build the crawler
RUN go build -o crawler ./cmd/crawler

# Command to run when the container starts
CMD ["./crawler"]