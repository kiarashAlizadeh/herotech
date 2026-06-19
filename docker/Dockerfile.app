# Stage 1: Build the application
# Use an official Go runtime as a parent image. This image includes the Go compiler and build tools.
FROM golang:1.25.5-alpine3.23 AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum to download dependencies.
# This step is cached, so dependencies are only downloaded when they change.
COPY go.mod go.sum ./

# Download all dependencies.
RUN go mod download

# Copy the source code of the application.
COPY . .

# Build the Go application.
# CGO_ENABLED=0: Disables CGO, creating a static binary without C dependencies.
# GOOS=linux: Ensures the binary is built for Linux.
# -ldflags="-w -s": Strips debug information (-w) and symbol table (-s) to reduce binary size.
# -o /app/main: Specifies the output file name for the executable.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/main ./cmd/api/main.go

# Stage 2: Create the final, minimal image
# We Use a lightweight base image like Alpine Linux.
FROM alpine:3.23

# Install CA certificates for secure HTTPS connections.
RUN apk --no-cache add ca-certificates

# Set the working directory in the final image.
WORKDIR /root/

# Copy the compiled binary from the builder stage.
COPY --from=builder /app/main .

# Expose the port that application listens on.
EXPOSE 8080

# Command to run the executable when the container starts.
CMD ["./main"]
