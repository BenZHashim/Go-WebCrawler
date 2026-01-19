# Start with a small, lightweight Go image
FROM golang:1.24-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy ALL your new files (cmd/, internal/, etc.)
COPY . .

# FIX: Build the specific path where main.go lives
# Was: RUN go build -o crawler .
RUN go build -o crawler ./cmd/crawler

# Command to run when the container starts
CMD ["./crawler"]