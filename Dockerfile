# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build binary
RUN go build -o tinyurl .

# Stage 2: Run
FROM alpine:latest

WORKDIR /app

# Copy binary from stage builder
COPY --from=builder /app/tinyurl .

# Use port 7860 for Hugging Space
ENV PORT=7860
EXPOSE 7860

# Run application
CMD ["./tinyurl"]