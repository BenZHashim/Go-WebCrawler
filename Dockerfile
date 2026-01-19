# Start with a small, lightweight Go image
# Start with a small, lightweight Go image
FROM golang:1.24-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy ALL your new files (main.go, worker.go, etc.)
COPY . .

# Build everything in the current folder
RUN go build -o crawler .

# Command to run when the container starts
CMD ["./crawler"]