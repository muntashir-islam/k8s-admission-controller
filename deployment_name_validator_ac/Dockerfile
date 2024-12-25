# Start from the official Go image as the build stage
FROM golang:1.23.4 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o admission-webhook .

# Use a minimal image for the final build
FROM alpine:latest

# Copy the compiled binary from the builder
COPY --from=builder /app/admission-webhook /usr/local/bin/admission-webhook

# Expose port 8443 for the webhook server
EXPOSE 8443

# Command to run the executable
ENTRYPOINT ["/usr/local/bin/admission-webhook"]