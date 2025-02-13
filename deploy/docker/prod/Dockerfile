FROM golang:latest AS build

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY ../go.* ./
RUN go mod download

COPY ../api/ ./api
COPY ../internal/ ./internal
COPY ../ui/ ./ui
COPY main.go ./main.go

RUN CGO_ENABLED=0 GOOS=linux go build -o /api

FROM alpine:latest

# install ffmpeg
RUN apk update
RUN apk upgrade
RUN apk add --no-cache ffmpeg

COPY --from=build ./api /app/api

EXPOSE 443

CMD ["/app/api", "-maximum-number-phrases=500", "-env=prod"]
