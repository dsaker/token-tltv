# Stage 1: Build the application
FROM golang:1.23.8 AS builder
WORKDIR /app

# Install dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .
# Build for Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Stage 2: Create the final image
FROM alpine:3.19
# install ffmpeg
RUN apk update
RUN apk upgrade
RUN apk add --no-cache ffmpeg

# Copy the binary from builder
COPY --from=builder /app/app .

EXPOSE 8080

ENTRYPOINT ["/app"]
CMD ["-maximum-number-phrases=10"]
