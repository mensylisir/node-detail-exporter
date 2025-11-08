# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set the working directory
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Build the Go application
# The output binary will be named 'node-prober'
RUN CGO_ENABLED=1 go build -o /node-prober .

# Stage 2: Create the final lightweight image
FROM alpine:latest

# Install runtime dependencies for the collectors that need them
# mtr is needed for the mtr collector
RUN apk add --no-cache mtr

# Copy the compiled binary from the builder stage
COPY --from=builder /node-prober /usr/local/bin/node-prober

# Copy the configuration files
COPY configs /configs

# Set the entrypoint for the container
ENTRYPOINT ["/usr/local/bin/node-prober"]

# Default command can be to show help, or run with a default config
# For now, we'll assume it needs a config file path as an argument.
CMD ["--config.file=/configs/config.example.yaml"]
