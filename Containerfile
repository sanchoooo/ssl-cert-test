# Use the official Go image as the base image
FROM golang:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# --- OPTIMIZATION START ---
# 1. Copy ONLY the dependency files first. 
COPY go.mod go.sum ./

# 2. Download dependencies.
#    Docker will CACHE this step. It will NOT run again unless go.mod or go.sum changes.
RUN go mod download

# 3. NOW copy the actual source code.
COPY . .
# --- OPTIMIZATION END ---

# 4. Build the binary
#    (Removed 'go mod tidy' from here - see note below)
RUN CGO_ENABLED=0 GOOS=linux go build -o /executable ./cmd/ssl_cert_checker

# --- FINAL STAGE ---
FROM scratch

# Copy the binary into the final stage
COPY --from=builder /executable /executable
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 1001
# Launch the binary when the container starts
ENTRYPOINT [ "/executable" ]