# ---- Build Stage ----
# Use the official Golang image to build the application
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first,
# leveraging Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application, creating a static binary.
# CGO_ENABLED=0 is important for creating a truly static binary.
# -ldflags "-w -s" strips debugging information to reduce size.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s" -o /proxy .

# ---- Final Stage ----
# Use a minimal, non-root image for the final container.
# "distroless" images contain only the application and its runtime dependencies.
FROM gcr.io/distroless/static-debian12

# Set the working directory
WORKDIR /

# Copy the compiled binary from the builder stage
COPY --from=builder /proxy /proxy
COPY --from=builder /app/sa.json /app/sa.json

# Set the binary as the container's entrypoint
ENTRYPOINT ["/proxy"]
